use std::cell::RefCell;
use std::sync::Arc;

use resvg::tiny_skia::{Pixmap, Transform};
use resvg::usvg::{self, fontdb};

const FONTS: [&[u8]; 6] = [
    include_bytes!("../fonts/Inter-Regular.ttf"),
    include_bytes!("../fonts/Inter-Medium.ttf"),
    include_bytes!("../fonts/Inter-SemiBold.ttf"),
    include_bytes!("../fonts/Inter-Bold.ttf"),
    include_bytes!("../fonts/JetBrainsMono-Regular.ttf"),
    include_bytes!("../fonts/JetBrainsMono-Medium.ttf"),
];

thread_local! {
    static RESULT: RefCell<Vec<u8>> = const { RefCell::new(Vec::new()) };
    static FONTDB: RefCell<Option<Arc<fontdb::Database>>> = const { RefCell::new(None) };
}

fn database() -> Arc<fontdb::Database> {
    FONTDB.with(|cell| {
        cell.borrow_mut()
            .get_or_insert_with(|| {
                let mut db = fontdb::Database::new();
                for font in FONTS {
                    db.load_font_source(fontdb::Source::Binary(Arc::new(font)));
                }
                db.set_sans_serif_family("Inter");
                db.set_serif_family("Inter");
                db.set_monospace_family("JetBrains Mono");
                Arc::new(db)
            })
            .clone()
    })
}

fn draw(svg: &[u8], scale: f32) -> Result<Vec<u8>, String> {
    let options = usvg::Options {
        font_family: "Inter".to_string(),
        fontdb: database(),
        ..usvg::Options::default()
    };
    let tree = usvg::Tree::from_data(svg, &options).map_err(|e| e.to_string())?;
    let size = tree.size().to_int_size().scale_by(scale).ok_or("invalid scale")?;
    let mut pixmap = Pixmap::new(size.width(), size.height()).ok_or("invalid image size")?;
    resvg::render(&tree, Transform::from_scale(scale, scale), &mut pixmap.as_mut());
    pixmap.encode_png().map_err(|e| e.to_string())
}

#[unsafe(no_mangle)]
pub extern "C" fn alloc(len: usize) -> *mut u8 {
    let mut buf = Vec::<u8>::with_capacity(len.max(1));
    let ptr = buf.as_mut_ptr();
    std::mem::forget(buf);
    ptr
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn dealloc(ptr: *mut u8, len: usize) {
    drop(unsafe { Vec::from_raw_parts(ptr, 0, len.max(1)) });
}

#[unsafe(no_mangle)]
pub unsafe extern "C" fn render(svg_ptr: *const u8, svg_len: usize, scale: f32) -> i32 {
    let svg = unsafe { std::slice::from_raw_parts(svg_ptr, svg_len) };
    let (code, out) = match draw(svg, scale) {
        Ok(png) => (0, png),
        Err(msg) => (1, msg.into_bytes()),
    };
    RESULT.with(|r| *r.borrow_mut() = out);
    code
}

#[unsafe(no_mangle)]
pub extern "C" fn result_ptr() -> *const u8 {
    RESULT.with(|r| r.borrow().as_ptr())
}

#[unsafe(no_mangle)]
pub extern "C" fn result_len() -> usize {
    RESULT.with(|r| r.borrow().len())
}
