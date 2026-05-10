zephyr_library()

zephyr_include_directories(../zmk/app/include)
zephyr_library_sources_ifdef(CONFIG_ZMK_SPLIT_ROLE_CENTRAL ../src/layer_ble_service.c)
