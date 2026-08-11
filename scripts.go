package main

import _ "embed"

//go:embed tmpl/header.js.tmpl
var JS_HEADER string

//go:embed tmpl/window_list_helpers.js.tmpl
var JS_WINDOW_LIST_HELPERS string

//go:embed tmpl/tile_unassignment_helper.js.tmpl
var JS_TILE_UNASSIGNMENT_HELPER string

//go:embed tmpl/tile_helpers.js.tmpl
var JS_TILE_HELPERS string

//go:embed tmpl/list.js.tmpl
var JS_LIST string

//go:embed tmpl/find.js.tmpl
var JS_FIND string

//go:embed tmpl/get_active_window.js.tmpl
var JS_GET_ACTIVE_WINDOW string

//go:embed tmpl/get_window_geometry.js.tmpl
var JS_GET_WINDOW_GEOMETRY string

//go:embed tmpl/get_workspace.js.tmpl
var JS_GET_WORKSPACE string

//go:embed tmpl/set_workspace.js.tmpl
var JS_SET_WORKSPACE string

//go:embed tmpl/activate_window.js.tmpl
var JS_ACTIVATE_WINDOW string

//go:embed tmpl/set_window_size.js.tmpl
var JS_SET_WINDOW_SIZE string

//go:embed tmpl/set_window_position.js.tmpl
var JS_SET_WINDOW_POSITION string

//go:embed tmpl/set_window_geometry.js.tmpl
var JS_SET_WINDOW_GEOMETRY string

//go:embed tmpl/set_window_workspace.js.tmpl
var JS_SET_WINDOW_WORKSPACE string

//go:embed tmpl/set_window_property.js.tmpl
var JS_SET_WINDOW_PROPERTY string

//go:embed tmpl/close_window.js.tmpl
var JS_CLOSE_WINDOW string

//go:embed tmpl/mouse_pos.js.tmpl
var JS_MOUSE_POS string

//go:embed tmpl/list_outputs.js.tmpl
var JS_LIST_OUTPUTS string

//go:embed tmpl/get_active_output.js.tmpl
var JS_GET_ACTIVE_OUTPUT string

//go:embed tmpl/get_cursor_output.js.tmpl
var JS_GET_CURSOR_OUTPUT string

//go:embed tmpl/get_active_window_output.js.tmpl
var JS_GET_ACTIVE_WINDOW_OUTPUT string

//go:embed tmpl/get_output_geometry.js.tmpl
var JS_GET_OUTPUT_GEOMETRY string

//go:embed tmpl/get_window_opacity.js.tmpl
var JS_GET_WINDOW_OPACITY string

//go:embed tmpl/set_window_opacity.js.tmpl
var JS_SET_WINDOW_OPACITY string

//go:embed tmpl/increase_window_opacity.js.tmpl
var JS_INCREASE_WINDOW_OPACITY string

//go:embed tmpl/decrease_window_opacity.js.tmpl
var JS_DECREASE_WINDOW_OPACITY string

//go:embed tmpl/set_window_geometry_relative.js.tmpl
var JS_SET_WINDOW_GEOMETRY_RELATIVE string

//go:embed tmpl/set_window_size_relative.js.tmpl
var JS_SET_WINDOW_SIZE_RELATIVE string

//go:embed tmpl/set_window_position_relative.js.tmpl
var JS_SET_WINDOW_POSITION_RELATIVE string

//go:embed tmpl/list_tiles.js.tmpl
var JS_LIST_TILES string

//go:embed tmpl/get_window_tile.js.tmpl
var JS_GET_WINDOW_TILE string

//go:embed tmpl/set_window_tile.js.tmpl
var JS_SET_WINDOW_TILE string

//go:embed tmpl/unset_window_tile.js.tmpl
var JS_UNSET_WINDOW_TILE string

//go:embed tmpl/list_tile_windows.js.tmpl
var JS_LIST_TILE_WINDOWS string

//go:embed tmpl/footer.js.tmpl
var JS_FOOTER string
