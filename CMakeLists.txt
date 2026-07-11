cmake_minimum_required(VERSION 3.16)
project(TimePlanner VERSION 3.0.0 LANGUAGES CXX)

set(CMAKE_CXX_STANDARD 17)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_CXX_EXTENSIONS OFF)

include(FetchContent)

FetchContent_Declare(
    webview
    GIT_REPOSITORY https://github.com/webview/webview.git
    GIT_TAG 0.12.0
    GIT_SHALLOW TRUE
)
FetchContent_MakeAvailable(webview)

set(TIME_PLANNER_SOURCES
    client/windows/src/main.cpp
)
if(WIN32)
    list(APPEND TIME_PLANNER_SOURCES client/windows/resources/app.rc)
endif()

add_executable(time_planner ${TIME_PLANNER_SOURCES})
if(WIN32)
    set_target_properties(time_planner PROPERTIES WIN32_EXECUTABLE TRUE)
endif()

target_link_libraries(time_planner PRIVATE webview::core)

if(MSVC)
    target_compile_options(time_planner PRIVATE /W4 /utf-8)
    target_link_options(time_planner PRIVATE /ENTRY:mainCRTStartup)
else()
    target_compile_options(time_planner PRIVATE -Wall -Wextra -Wpedantic)
endif()
