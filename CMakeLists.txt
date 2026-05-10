zephyr_library()

zephyr_library_sources_ifdef(CONFIG_ZMK_WIDGET_NYAN src/nyan_widget.c)
zephyr_library_sources(src/layer_ble_service.c)
