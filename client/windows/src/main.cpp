#include <cstdlib>
#include <memory>
#include <string>

#include <webview.h>

namespace {

std::string appUrl() {
#ifdef _WIN32
    char* raw = nullptr;
    std::size_t len = 0;
    if (_dupenv_s(&raw, &len, "CHRONA_URL") == 0 && raw && raw[0] != '\0') {
        std::unique_ptr<char, decltype(&std::free)> holder(raw, std::free);
        return holder.get();
    }
    std::free(raw);
#else
    const char* configured = std::getenv("CHRONA_URL");
    if (configured && configured[0] != '\0') {
        return configured;
    }
#endif
    return "http://localhost:8765";
}

}  // namespace

int main() {
    webview::webview app(true, nullptr);
    app.set_title(u8"Chrona 时序");
    app.set_size(1200, 800, WEBVIEW_HINT_MIN);
    app.navigate(appUrl());
    app.run();
    return 0;
}
