package config

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

const (
	LayoutCard = "card"
	LayoutWide = "wide"

	DefaultBasePath = "ghstats/"

	LinkStyleFull = "full"
	LinkStyleIcon = "icon"
	LinkStyleText = "text"

	KindCustom = "custom"

	PartHeader   = "header"
	PartName     = "name"
	PartTagline  = "tagline"
	PartLocation = "location"
	PartNote     = "note"
	PartBuilding = "building"
	PartFocus    = "focus"
	PartFooter   = "footer"

	maxFocus    = 6
	maxBuilding = 4
)

var identityParts = []string{PartHeader, PartName, PartTagline, PartLocation, PartNote, PartBuilding, PartFocus, PartFooter}

type Profile struct {
	Name     string  `yaml:"name,omitempty"`
	Tagline  string  `yaml:"tagline,omitempty"`
	Location string  `yaml:"location,omitempty"`
	Note     string  `yaml:"note,omitempty"`
	Focus    []Focus `yaml:"focus,omitempty"`
	Links    []Link  `yaml:"links,omitempty"`

	LinkStyle string          `yaml:"link_style,omitempty"`
	Identity  IdentityOptions `yaml:"identity,omitempty"`
}

type IdentityOptions struct {
	Show   []string `yaml:"show,omitempty"`
	Layout string   `yaml:"layout,omitempty"`
}

func IdentityParts() []string { return slices.Clone(identityParts) }

func (o IdentityOptions) Shows(part string) bool {
	return len(o.Show) == 0 || slices.Contains(o.Show, part)
}

type Focus struct {
	Name     string   `yaml:"name"`
	Building []string `yaml:"building"`
}

type Link struct {
	Kind  string
	Value string
	Label string
	Icon  string
	Style string
}

type ResolvedLink struct {
	Kind  string
	URL   string
	Label string
	Icon  string
	Style string
}

type linkKind struct {
	handleURL string
	label     func(h string) string
	urlOnly   bool
	mapOnly   bool
}

func fixed(s string) func(string) string { return func(string) string { return s } }
func prefixed(p string) func(string) string {
	return func(h string) string { return p + h }
}

var linkKinds = map[string]linkKind{
	"youtube":         {handleURL: "https://www.youtube.com/@%s", label: prefixed("@")},
	"twitch":          {handleURL: "https://www.twitch.tv/%s", label: prefixed("")},
	"kick":            {handleURL: "https://kick.com/%s", label: prefixed("")},
	"discord":         {handleURL: "https://discord.gg/%s", label: fixed("discord")},
	"tiktok":          {handleURL: "https://www.tiktok.com/@%s", label: prefixed("@")},
	"instagram":       {handleURL: "https://www.instagram.com/%s", label: prefixed("@")},
	"x":               {handleURL: "https://x.com/%s", label: prefixed("@")},
	"bluesky":         {handleURL: "https://bsky.app/profile/%s", label: prefixed("")},
	"threads":         {handleURL: "https://www.threads.net/@%s", label: prefixed("@")},
	"mastodon":        {urlOnly: true},
	"reddit":          {handleURL: "https://www.reddit.com/user/%s", label: prefixed("u/")},
	"linkedin":        {handleURL: "https://www.linkedin.com/in/%s", label: prefixed("")},
	"steam":           {urlOnly: true, label: fixed("steam")},
	"itchio":          {handleURL: "https://%s.itch.io", label: prefixed("")},
	"website":         {urlOnly: true},
	"github_sponsors": {handleURL: "https://github.com/sponsors/%s", label: fixed("sponsor")},
	"open_collective": {handleURL: "https://opencollective.com/%s", label: prefixed("")},
	"buy_me_a_coffee": {handleURL: "https://buymeacoffee.com/%s", label: fixed("coffee")},
	"ko_fi":           {handleURL: "https://ko-fi.com/%s", label: fixed("ko-fi")},
	"patreon":         {handleURL: "https://www.patreon.com/%s", label: prefixed("")},
	KindCustom:        {urlOnly: true, mapOnly: true},
}

func LinkKinds() []string { return sortedKeys(linkKinds) }

func LinkKindIsURLOnly(kind string) bool { return linkKinds[kind].urlOnly }

func LinkKindIsMapOnly(kind string) bool { return linkKinds[kind].mapOnly }

func LinkStyles() []string { return []string{LinkStyleFull, LinkStyleIcon, LinkStyleText} }

func IsIconPath(s string) bool {
	if !strings.HasPrefix(s, "M") && !strings.HasPrefix(s, "m") {
		return false
	}
	return strings.IndexFunc(s, unicode.IsControl) < 0 && !strings.ContainsAny(s, "<>\"&")
}

func FocusLimits() (maxEntries, maxWords int) { return maxFocus, maxBuilding }

func (l *Link) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.MappingNode || len(n.Content) != 2 {
		return fmt.Errorf("line %d: a link item must be a map with exactly one key (kind: handle or url)", n.Line)
	}
	kind, val := n.Content[0].Value, n.Content[1]
	switch val.Kind {
	case yaml.ScalarNode:
		if kind == KindCustom {
			return fmt.Errorf("line %d: link %q needs the map form, for example custom: {url: \"https://example.com\", label: \"my blog\", icon: \"github\"}", val.Line, kind)
		}
		*l = Link{Kind: kind, Value: val.Value}
		return nil
	case yaml.MappingNode:
		out := Link{Kind: kind}
		haveURL := false
		for i := 0; i < len(val.Content); i += 2 {
			k, v := val.Content[i].Value, val.Content[i+1]
			if v.Kind != yaml.ScalarNode {
				return fmt.Errorf("line %d: link %q: %s must be a string", v.Line, kind, k)
			}
			switch k {
			case "url":
				out.Value, haveURL = v.Value, true
			case "label":
				out.Label = v.Value
			case "style":
				out.Style = v.Value
			case "icon":
				if kind != KindCustom {
					return fmt.Errorf("line %d: link %q: the icon key is only valid for the %q kind", val.Content[i].Line, kind, KindCustom)
				}
				out.Icon = v.Value
			default:
				want := "url, label and style"
				if kind == KindCustom {
					want = "url, label, icon and style"
				}
				return fmt.Errorf("line %d: link %q: unknown key %q (want %s)", val.Content[i].Line, kind, k, want)
			}
		}
		if !haveURL {
			return fmt.Errorf("line %d: link %q: the map form needs a url", val.Line, kind)
		}
		*l = out
		return nil
	}
	return fmt.Errorf("line %d: link %q must be a handle, a URL or a {url, label} map", val.Line, kind)
}

func (l Link) MarshalYAML() (any, error) {
	if l.Label == "" && l.Icon == "" && l.Style == "" && l.Kind != KindCustom {
		return map[string]string{l.Kind: l.Value}, nil
	}
	m := map[string]string{"url": l.Value}
	if l.Label != "" {
		m["label"] = l.Label
	}
	if l.Icon != "" {
		m["icon"] = l.Icon
	}
	if l.Style != "" {
		m["style"] = l.Style
	}
	return map[string]map[string]string{l.Kind: m}, nil
}

func (l Link) Empty() bool { return strings.TrimSpace(l.Value) == "" }

func (p Profile) HasIdentity() bool {
	return len(p.Focus) > 0 || strings.TrimSpace(p.Tagline) != ""
}

func (p Profile) ResolvedLinks() []ResolvedLink {
	var out []ResolvedLink
	for _, l := range p.Links {
		if l.Empty() {
			continue
		}
		u, label, err := l.Resolve()
		if err != nil {
			continue
		}
		style := p.LinkStyle
		if l.Style != "" {
			style = l.Style
		}
		out = append(out, ResolvedLink{Kind: l.Kind, URL: u, Label: label, Icon: l.Icon, Style: style})
	}
	return out
}

func (l Link) Resolve() (string, string, error) {
	k, ok := linkKinds[l.Kind]
	if !ok {
		return "", "", fmt.Errorf("unknown link kind %q (want %s)", l.Kind, strings.Join(LinkKinds(), ", "))
	}
	v := strings.TrimSpace(l.Value)
	if v == "" {
		return "", "", fmt.Errorf("link %q: empty value", l.Kind)
	}
	if err := l.checkCustom(); err != nil {
		return "", "", err
	}
	var link, label string
	if strings.Contains(v, "://") || strings.HasPrefix(strings.ToLower(v), "http:") {
		u, err := url.Parse(v)
		switch {
		case err != nil:
			return "", "", fmt.Errorf("link %q: invalid URL %q: %v", l.Kind, v, err)
		case u.Scheme != "https":
			return "", "", fmt.Errorf("link %q: URL must start with https://, got %q", l.Kind, v)
		case u.Host == "":
			return "", "", fmt.Errorf("link %q: URL %q has no host", l.Kind, v)
		}
		link = v
		label = urlLabel(l.Kind, k, u)
	} else {
		if k.urlOnly {
			return "", "", fmt.Errorf("link %q: needs a full https:// URL, got %q", l.Kind, v)
		}
		if strings.IndexFunc(v, unicode.IsSpace) >= 0 || strings.ContainsAny(v, "<>\"") {
			return "", "", fmt.Errorf("link %q: invalid handle %q", l.Kind, v)
		}
		h := strings.TrimPrefix(v, "@")
		if h == "" {
			return "", "", fmt.Errorf("link %q: invalid handle %q", l.Kind, v)
		}
		link = fmt.Sprintf(k.handleURL, h)
		label = k.label(h)
	}
	if l.Label != "" {
		label = l.Label
	}
	return link, label, nil
}

func (l Link) checkCustom() error {
	if l.Kind != KindCustom {
		if l.Icon != "" {
			return fmt.Errorf("link %q: the icon key is only valid for the %q kind", l.Kind, KindCustom)
		}
		return nil
	}
	if strings.TrimSpace(l.Label) == "" {
		return fmt.Errorf("link %q: needs a label", l.Kind)
	}
	switch {
	case l.Icon == "", IsIconKind(l.Icon):
	case strings.HasPrefix(l.Icon, "M") || strings.HasPrefix(l.Icon, "m"):
		if !IsIconPath(l.Icon) {
			return fmt.Errorf("link %q: icon path must not contain <, >, \" or &", l.Kind)
		}
	default:
		return fmt.Errorf("link %q: icon %q is neither a link kind (%s) nor an SVG path starting with M", l.Kind, l.Icon, strings.Join(iconKinds(), ", "))
	}
	return nil
}

func IsIconKind(s string) bool {
	_, ok := linkKinds[s]
	return ok && s != KindCustom
}

func iconKinds() []string {
	return slices.DeleteFunc(LinkKinds(), func(k string) bool { return k == KindCustom })
}

func urlLabel(kind string, k linkKind, u *url.URL) string {
	host := u.Host
	if k.urlOnly {
		if k.label != nil {
			return k.label("")
		}
		if kind == "mastodon" {
			return host + strings.TrimRight(u.Path, "/")
		}
		return host
	}
	h := strings.TrimPrefix(lastSegment(u.Path), "@")
	if kind == "itchio" {
		h = strings.TrimSuffix(host, ".itch.io")
	}
	if h == "" {
		h = host
	}
	return k.label(h)
}

func lastSegment(p string) string {
	parts := strings.Split(strings.Trim(p, "/"), "/")
	return parts[len(parts)-1]
}

func (p Profile) validate(fail func(format string, args ...any)) {
	if p.LinkStyle != "" && !slices.Contains(LinkStyles(), p.LinkStyle) {
		fail("profile.link_style: must be one of %s, got %q", strings.Join(LinkStyles(), ", "), p.LinkStyle)
	}
	seenPart := map[string]bool{}
	for i, part := range p.Identity.Show {
		switch {
		case !slices.Contains(identityParts, part):
			fail("profile.identity.show[%d]: unknown part %q (want %s)", i, part, strings.Join(identityParts, ", "))
		case seenPart[part]:
			fail("profile.identity.show[%d]: part %q is listed more than once", i, part)
		}
		seenPart[part] = true
	}
	if p.Identity.Layout != "" && !validLayout(p.Identity.Layout) {
		fail("profile.identity.layout: must be %q or %q, got %q", LayoutWide, LayoutCard, p.Identity.Layout)
	}
	if p.Focus != nil && (len(p.Focus) < 1 || len(p.Focus) > maxFocus) {
		fail("profile.focus: must have 1 to %d entries, got %d", maxFocus, len(p.Focus))
	}
	for i, f := range p.Focus {
		if strings.TrimSpace(f.Name) == "" {
			fail("profile.focus[%d]: name must not be empty", i)
		}
		if len(f.Building) < 1 || len(f.Building) > maxBuilding {
			fail("profile.focus[%d]: building must have 1 to %d words, got %d", i, maxBuilding, len(f.Building))
		}
		for j, w := range f.Building {
			if strings.TrimSpace(w) == "" {
				fail("profile.focus[%d].building[%d]: must not be empty", i, j)
			}
		}
	}
	seen := map[string]bool{}
	for i, l := range p.Links {
		if _, ok := linkKinds[l.Kind]; !ok {
			fail("profile.links[%d]: unknown kind %q (want %s)", i, l.Kind, strings.Join(LinkKinds(), ", "))
			continue
		}
		if seen[l.Kind] {
			fail("profile.links[%d]: kind %q is listed more than once", i, l.Kind)
		}
		seen[l.Kind] = true
		if l.Style != "" && !slices.Contains(LinkStyles(), l.Style) {
			fail("profile.links[%d].style: must be one of %s, got %q", i, strings.Join(LinkStyles(), ", "), l.Style)
		}
		if l.Empty() {
			continue
		}
		if _, _, err := l.Resolve(); err != nil {
			fail("profile.links[%d]: %v", i, err)
		}
	}
}

func validLayout(s string) bool { return slices.Contains([]string{LayoutCard, LayoutWide}, s) }

func validBasePath(s string) bool {
	if s == "" || !strings.HasSuffix(s, "/") || !utf8.ValidString(s) {
		return false
	}
	if strings.ContainsAny(s, "\\<>\"'`") || strings.Contains(s, "..") {
		return false
	}
	return strings.IndexFunc(s, unicode.IsSpace) < 0
}
