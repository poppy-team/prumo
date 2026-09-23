use freya::icons::lucide;
use freya::prelude::Bytes;

/// Retrieves Lucide SVG icon by name
pub fn icon(name: &str) -> Bytes {
    match name {
        "folder" => lucide::folder(),
        "file_code_2" => lucide::file_code_2(),
        "file_code" => lucide::file_code(),
        "package" => lucide::package(),
        "file_text" => lucide::file_text(),
        "git_branch" => lucide::git_branch(),
        "circle_alert" => lucide::circle_alert(),
        "triangle_alert" => lucide::triangle_alert(),
        "files" => lucide::files(),
        "bot" => lucide::bot(),
        "chevron_right" => lucide::chevron_right(),
        "chevron_down" => lucide::chevron_down(),
        "search" => lucide::search(),
        "x" => lucide::x(),
        "save" => lucide::save(),
        "refresh" => lucide::refresh_cw(),
        "settings" => lucide::settings(),
        "terminal" => lucide::terminal(),
        "play" => lucide::play(),
        "stop" => lucide::circle_stop(),
        "list_checks" => lucide::list_checks(),
        "panel_bottom" => lucide::panel_bottom(),
        _ => lucide::file(),
    }
}
