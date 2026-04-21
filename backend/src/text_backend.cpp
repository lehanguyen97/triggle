#include <e/backend_api.h>

#include <stdlib.h>
#include <string.h>

#include <string>
#include <vector>

#include <ft2build.h>
#include FT_FREETYPE_H
#include <hb.h>
#include <hb-ft.h>

struct BackendTextFont {
    FT_Library library = nullptr;
    FT_Face face = nullptr;
    hb_font_t* hb_font = nullptr;
};

static std::vector<BackendTextFont*> g_text_fonts;

static BackendTextFont* get_text_font(text_font_t handle) {
    if (handle < 0 || handle >= (text_font_t)g_text_fonts.size()) {
        return nullptr;
    }
    return g_text_fonts[(size_t)handle];
}

static text_font_t store_text_font(BackendTextFont* font) {
    for (size_t i = 0; i < g_text_fonts.size(); i++) {
        if (g_text_fonts[i] == nullptr) {
            g_text_fonts[i] = font;
            return (text_font_t)i;
        }
    }
    g_text_fonts.push_back(font);
    return (text_font_t)(g_text_fonts.size() - 1);
}

static void destroy_text_font(BackendTextFont* font) {
    if (!font) {
        return;
    }
    if (font->hb_font) {
        hb_font_destroy(font->hb_font);
    }
    if (font->face) {
        FT_Done_Face(font->face);
    }
    if (font->library) {
        FT_Done_FreeType(font->library);
    }
    free(font);
}

extern "C" {

text_font_t backend_text_font_open(backend_t e, const char* path, int32_t path_len, float pt_size) {
    if (e != 0 || !path || path_len <= 0 || pt_size <= 0.0f) {
        return -1;
    }

    std::string font_path(path, (size_t)path_len);
    BackendTextFont* font = (BackendTextFont*)calloc(1, sizeof(BackendTextFont));
    if (!font) {
        return -1;
    }
    if (FT_Init_FreeType(&font->library) != 0) {
        destroy_text_font(font);
        return -1;
    }
    if (FT_New_Face(font->library, font_path.c_str(), 0, &font->face) != 0) {
        destroy_text_font(font);
        return -1;
    }
    FT_F26Dot6 pts = (FT_F26Dot6)(pt_size * 64.0f);
    if (FT_Set_Char_Size(font->face, 0, pts, 0, 0) != 0) {
        destroy_text_font(font);
        return -1;
    }
    font->hb_font = hb_ft_font_create(font->face, nullptr);
    if (!font->hb_font) {
        destroy_text_font(font);
        return -1;
    }
    return store_text_font(font);
}

void backend_text_font_close(backend_t e, text_font_t font) {
    if (e != 0) {
        return;
    }
    BackendTextFont* state = get_text_font(font);
    if (!state) {
        return;
    }
    g_text_fonts[(size_t)font] = nullptr;
    destroy_text_font(state);
}

int32_t backend_text_font_get_metrics(backend_t e, text_font_t font, backend_text_metrics_t* out_metrics) {
    if (!out_metrics) {
        return -1;
    }
    memset(out_metrics, 0, sizeof(*out_metrics));
    if (e != 0) {
        return -1;
    }
    BackendTextFont* state = get_text_font(font);
    if (!state || !state->face || !state->face->size) {
        return -1;
    }
    FT_Size_Metrics m = state->face->size->metrics;
    out_metrics->height_px = (int32_t)(m.height >> 6);
    out_metrics->ascent_px = (int32_t)(m.ascender >> 6);
    out_metrics->descent_px = (int32_t)(-(m.descender >> 6));
    out_metrics->line_skip_px = (int32_t)(m.height >> 6);
    return 0;
}

int32_t backend_text_measure_utf8(backend_t backend,
                                  text_font_t font,
                                  const void* utf8,
                                  int32_t utf8_len,
                                  backend_text_measure_t* out_measure) {
    if (!out_measure) {
        return -1;
    }
    memset(out_measure, 0, sizeof(*out_measure));
    if (backend != 0) {
        return -1;
    }
    backend_text_shape_info_t shape;
    int32_t r = backend_text_shape_utf8(backend, font, utf8, utf8_len, nullptr, 0, &shape);
    if (r != 0) {
        return r;
    }
    out_measure->width_26_6 = shape.width_26_6;
    out_measure->height_26_6 = shape.height_26_6;
    return 0;
}

int32_t backend_text_shape_utf8(backend_t e,
                                text_font_t font,
                                const void* utf8,
                                int32_t utf8_len,
                                backend_text_shaped_glyph_t* out_glyphs,
                                int32_t glyph_cap,
                                backend_text_shape_info_t* out_shape) {
    if (!out_shape) {
        return -1;
    }
    memset(out_shape, 0, sizeof(*out_shape));
    if (e != 0) {
        return -1;
    }
    BackendTextFont* state = get_text_font(font);
    if (!state || !state->hb_font || !state->face) {
        return -1;
    }
    if (utf8_len < 0) {
        utf8_len = utf8 ? (int32_t)strlen((const char*)utf8) : 0;
    }
    if (!utf8 || utf8_len == 0) {
        return 0;
    }

    hb_buffer_t* buf = hb_buffer_create();
    if (!buf) {
        return -1;
    }
    hb_buffer_add_utf8(buf, (const char*)utf8, utf8_len, 0, utf8_len);
    hb_buffer_guess_segment_properties(buf);
    hb_shape(state->hb_font, buf, nullptr, 0);

    unsigned int glyph_count = 0;
    hb_glyph_info_t* infos = hb_buffer_get_glyph_infos(buf, &glyph_count);
    hb_glyph_position_t* pos = hb_buffer_get_glyph_positions(buf, &glyph_count);

    int32_t width = 0;
    for (unsigned int i = 0; i < glyph_count; i++) {
        width += pos[i].x_advance;
    }
    out_shape->glyph_count = (int32_t)glyph_count;
    out_shape->width_26_6 = width;
    out_shape->height_26_6 = (int32_t)state->face->size->metrics.height;

    if (!out_glyphs || glyph_cap <= 0) {
        hb_buffer_destroy(buf);
        return 0;
    }
    if ((unsigned int)glyph_cap < glyph_count) {
        hb_buffer_destroy(buf);
        return -1;
    }

    for (unsigned int i = 0; i < glyph_count; i++) {
        out_glyphs[i].glyph_id = infos[i].codepoint;
        out_glyphs[i].cluster = infos[i].cluster;
        out_glyphs[i].x_offset_26_6 = pos[i].x_offset;
        out_glyphs[i].y_offset_26_6 = pos[i].y_offset;
        out_glyphs[i].x_advance_26_6 = pos[i].x_advance;
        out_glyphs[i].y_advance_26_6 = pos[i].y_advance;
    }

    hb_buffer_destroy(buf);
    return 0;
}

int32_t backend_text_raster_glyph_rgba8(backend_t e,
                                        text_font_t font,
                                        uint32_t glyph_id,
                                        void* out_pixels,
                                        int32_t pixel_cap,
                                        backend_text_glyph_bitmap_t* out_bitmap) {
    if (!out_bitmap) {
        return -1;
    }
    memset(out_bitmap, 0, sizeof(*out_bitmap));
    out_bitmap->glyph_id = glyph_id;
    if (e != 0) {
        return -1;
    }
    BackendTextFont* state = get_text_font(font);
    if (!state || !state->face) {
        return -1;
    }
    if (FT_Load_Glyph(state->face, glyph_id, FT_LOAD_RENDER) != 0) {
        return -1;
    }

    FT_GlyphSlot slot = state->face->glyph;
    FT_Bitmap* bmp = &slot->bitmap;
    if (bmp->pixel_mode != FT_PIXEL_MODE_GRAY && !(bmp->width == 0 && bmp->rows == 0)) {
        return -1;
    }

    out_bitmap->width_px = (int32_t)bmp->width;
    out_bitmap->height_px = (int32_t)bmp->rows;
    out_bitmap->bearing_x_px = slot->bitmap_left;
    out_bitmap->bearing_y_px = slot->bitmap_top;
    out_bitmap->stride_bytes = (int32_t)bmp->width * 4;

    if (bmp->width == 0 || bmp->rows == 0) {
        return 0;
    }

    const int32_t needed = (int32_t)bmp->width * (int32_t)bmp->rows * 4;
    if (!out_pixels || pixel_cap <= 0) {
        return 0;
    }
    if (pixel_cap < needed) {
        return -1;
    }

    unsigned char* dst = (unsigned char*)out_pixels;
    memset(dst, 0, (size_t)needed);
    for (unsigned int row = 0; row < bmp->rows; row++) {
        for (unsigned int col = 0; col < bmp->width; col++) {
            unsigned char a = bmp->buffer[row * (unsigned int)bmp->pitch + col];
            size_t di = ((size_t)row * (size_t)bmp->width + (size_t)col) * 4;
            dst[di + 0] = 255;
            dst[di + 1] = 255;
            dst[di + 2] = 255;
            dst[di + 3] = a;
        }
    }
    return 0;
}

}  // extern "C"
