/*
 * Copyright (c) 2026
 *
 * SPDX-License-Identifier: MIT
 */

#include <string.h>
#include <errno.h>

#include <zephyr/bluetooth/gatt.h>
#include <zephyr/bluetooth/uuid.h>
#include <zephyr/init.h>
#include <zephyr/kernel.h>
#include <zephyr/logging/log.h>
#include <zephyr/sys/printk.h>

#include <zmk/event_manager.h>
#include <zmk/events/layer_state_changed.h>
#include <zmk/keymap.h>

LOG_MODULE_REGISTER(corne_layer_ble_service, CONFIG_ZMK_LOG_LEVEL);

#define CORNE_LAYER_SERVICE_UUID_VAL BT_UUID_128_ENCODE(0x4fafc201, 0x1fb5, 0x459e, 0x8fcc,       \
                                                        0xc5c9c331914b)
#define CORNE_LAYER_CHAR_UUID_VAL BT_UUID_128_ENCODE(0xbeb5483e, 0x36e1, 0x4688, 0xb7f5,          \
                                                     0xea07361b26a8)
#define CORNE_LAYER_MAX_NAME_LEN 32

static char current_layer_name[CORNE_LAYER_MAX_NAME_LEN] = "UNKNOWN";

static void corne_layer_update_name(void) {
    zmk_keymap_layer_index_t layer = zmk_keymap_highest_layer_active();
    const char *name = zmk_keymap_layer_name(layer);

    if (name == NULL || name[0] == '\0') {
        snprintk(current_layer_name, sizeof(current_layer_name), "LAYER %u", layer);
        return;
    }

    snprintk(current_layer_name, sizeof(current_layer_name), "%s", name);
}

static ssize_t corne_layer_read(struct bt_conn *conn, const struct bt_gatt_attr *attr, void *buf,
                                uint16_t len, uint16_t offset) {
    ARG_UNUSED(attr);

    corne_layer_update_name();
    return bt_gatt_attr_read(conn, attr, buf, len, offset, current_layer_name,
                             strlen(current_layer_name));
}

static void corne_layer_ccc_changed(const struct bt_gatt_attr *attr, uint16_t value) {
    ARG_UNUSED(attr);
    LOG_DBG("Layer notify %s", value == BT_GATT_CCC_NOTIFY ? "enabled" : "disabled");
}

BT_GATT_SERVICE_DEFINE(
    corne_layer_svc, BT_GATT_PRIMARY_SERVICE(BT_UUID_DECLARE_128(CORNE_LAYER_SERVICE_UUID_VAL)),
    BT_GATT_CHARACTERISTIC(BT_UUID_DECLARE_128(CORNE_LAYER_CHAR_UUID_VAL),
                           BT_GATT_CHRC_READ | BT_GATT_CHRC_NOTIFY, BT_GATT_PERM_READ_ENCRYPT,
                           corne_layer_read, NULL, current_layer_name),
    BT_GATT_CCC(corne_layer_ccc_changed, BT_GATT_PERM_READ_ENCRYPT | BT_GATT_PERM_WRITE_ENCRYPT), );

static int corne_layer_publish(const zmk_event_t *eh) {
    ARG_UNUSED(eh);

    corne_layer_update_name();

    int err = bt_gatt_notify(NULL, &corne_layer_svc.attrs[1], current_layer_name,
                             strlen(current_layer_name));
    if (err && err != -ENOTCONN) {
        LOG_DBG("Failed to notify active layer: %d", err);
    }

    return ZMK_EV_EVENT_BUBBLE;
}

static int corne_layer_service_init(void) {
    corne_layer_update_name();
    return 0;
}

SYS_INIT(corne_layer_service_init, APPLICATION, CONFIG_APPLICATION_INIT_PRIORITY);

ZMK_LISTENER(corne_layer_ble_service, corne_layer_publish);
ZMK_SUBSCRIPTION(corne_layer_ble_service, zmk_layer_state_changed);
