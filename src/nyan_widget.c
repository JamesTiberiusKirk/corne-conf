// SPDX-License-Identifier: MIT
#include <zephyr/kernel.h>
#include <zephyr/init.h>
#include <zephyr/device.h>
#include <zephyr/devicetree.h>
#include <zephyr/sys/util.h>
#include <lvgl.h>

/* Bind compatible for DT iteration */
#define DT_DRV_COMPAT zmk_widget_nyan

// Simple 1-bit placeholder frames (16x16) to keep binary small.
// You can replace these with real Nyan frames later.
// LVGL expects little-endian bitmaps for LV_IMG_CF_ALPHA_1BIT.
static const uint8_t nyan_frame_bits_0[] = {
    0x00,0x00,0x7E,0x00,0x81,0x00,0xA5,0x00,
    0x81,0x00,0xBD,0x00,0x81,0x00,0xA5,0x00,
    0x81,0x00,0x7E,0x00,0x00,0x00,0x18,0x00,
    0x3C,0x00,0x18,0x00,0x00,0x00
};
static const uint8_t nyan_frame_bits_1[] = {
    0x00,0x00,0x3C,0x00,0x42,0x00,0xA5,0x00,
    0x42,0x00,0x99,0x00,0x42,0x00,0xA5,0x00,
    0x42,0x00,0x3C,0x00,0x00,0x00,0x18,0x00,
    0x3C,0x00,0x18,0x00,0x00,0x00
};

static const lv_img_dsc_t nyan_frame_0 = {
    .header.always_zero = 0,
    .header.w = 16,
    .header.h = 16,
    .data_size = sizeof(nyan_frame_bits_0),
    .header.cf = LV_IMG_CF_ALPHA_1BIT,
    .data = nyan_frame_bits_0,
};
static const lv_img_dsc_t nyan_frame_1 = {
    .header.always_zero = 0,
    .header.w = 16,
    .header.h = 16,
    .data_size = sizeof(nyan_frame_bits_1),
    .header.cf = LV_IMG_CF_ALPHA_1BIT,
    .data = nyan_frame_bits_1,
};

struct nyan_instance {
    lv_obj_t *img;
    lv_timer_t *frame_timer;
    int frame_idx;
    int frame_count;
    int64_t start_ms;
    uint32_t timeout_ms;
};

static const lv_img_dsc_t *frames[] = { &nyan_frame_0, &nyan_frame_1 };

static void nyan_frame_cb(lv_timer_t *t)
{
    struct nyan_instance *inst = (struct nyan_instance *)t->user_data;
    if (!inst || !inst->img) { return; }

    /* Stop after timeout */
    if (inst->timeout_ms > 0) {
        int64_t now = k_uptime_get();
        if ((uint32_t)(now - inst->start_ms) >= inst->timeout_ms) {
            lv_timer_del(inst->frame_timer);
            inst->frame_timer = NULL;
            return;
        }
    }

    inst->frame_idx = (inst->frame_idx + 1) % inst->frame_count;
    lv_img_set_src(inst->img, frames[inst->frame_idx]);
}

/* For each DT node with compatible "zmk,widget-nyan", create an instance */
#define _INST(node_id)                                                                            \
    static struct nyan_instance inst_##node_id;                                                   \
    static void init_##node_id(struct k_work *work);                                              \
    K_WORK_DELAYABLE_DEFINE(work_##node_id, init_##node_id);                                      \
    static void init_##node_id(struct k_work *work) {                                             \
        ARG_UNUSED(work);                                                                         \
        int x = DT_PROP_OR(node_id, x, 80);                                                       \
        int y = DT_PROP_OR(node_id, y, 0);                                                        \
        uint32_t frame_delay_ms = DT_PROP_OR(node_id, frame_delay_ms, 100);                       \
        uint32_t timeout_ms = DT_PROP_OR(node_id, timeout_ms, 10000);                             \
        /* Create an image on the right display if present, else default */                       \
        inst_##node_id.frame_count = ARRAY_SIZE(frames);                                          \
        inst_##node_id.frame_idx = 0;                                                             \
        inst_##node_id.timeout_ms = timeout_ms;                                                   \
        inst_##node_id.start_ms = k_uptime_get();                                                 \
        lv_disp_t *disp_def = lv_disp_get_default();                                              \
        lv_disp_t *disp_target = lv_disp_get_next(disp_def); /* likely the right display */       \
        if (disp_target == NULL) {                                                                \
            disp_target = disp_def;                                                               \
        }                                                                                         \
        lv_obj_t *parent = lv_disp_get_scr_act(disp_target);                                      \
        inst_##node_id.img = lv_img_create(parent);                                               \
        lv_img_set_src(inst_##node_id.img, frames[0]);                                            \
        lv_obj_set_pos(inst_##node_id.img, x, y);                                                 \
        inst_##node_id.frame_timer = lv_timer_create(nyan_frame_cb, frame_delay_ms, &inst_##node_id); \
    }

/* Instantiate per-DT node objects/work items */
DT_FOREACH_STATUS_OKAY(DT_DRV_COMPAT, _INST);

/* Schedule the init a bit later to ensure LVGL/display are up */
static int nyan_init_all(void)
{
    /* 500 ms delay to let UI boot */
    #define SCHED(node_id) k_work_schedule(&work_##node_id, K_MSEC(500))
    DT_FOREACH_STATUS_OKAY(DT_DRV_COMPAT, SCHED);
    return 0;
}

SYS_INIT(nyan_init_all, APPLICATION, 99);
