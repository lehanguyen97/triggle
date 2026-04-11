#define CGLTF_IMPLEMENTATION
#include <cgltf.h>

#include "backend.hpp"

#include <vector>

static const int kPhongFloatsPerVert = 10; /* pos3 + normal3 + color4 */

int32_t Backend::gltf_load(const char* path) {
    cgltf_options options = {};
    cgltf_data* data = nullptr;
    cgltf_result r = cgltf_parse_file(&options, path, &data);
    if (r != cgltf_result_success || !data) return -1;
    r = cgltf_load_buffers(&options, data, path);
    if (r != cgltf_result_success) {
        cgltf_free(data);
        return -1;
    }

    GltfAsset ga;
    std::vector<mesh_t> created;

    auto rollback = [&]() {
        for (mesh_t m : created) mesh_destroy(m);
        created.clear();
    };

    for (cgltf_size mi = 0; mi < data->meshes_count; ++mi) {
        cgltf_mesh& mesh = data->meshes[mi];
        for (cgltf_size pi = 0; pi < mesh.primitives_count; ++pi) {
            cgltf_primitive& prim = mesh.primitives[pi];
            if (prim.type != cgltf_primitive_type_triangles) continue;

            const cgltf_accessor* pos = cgltf_find_accessor(&prim, cgltf_attribute_type_position, 0);
            if (!pos || pos->type != cgltf_type_vec3) {
                rollback();
                cgltf_free(data);
                return -1;
            }

            cgltf_size n = pos->count;
            const cgltf_accessor* nor_acc = cgltf_find_accessor(&prim, cgltf_attribute_type_normal, 0);
            const cgltf_accessor* col_acc = cgltf_find_accessor(&prim, cgltf_attribute_type_color, 0);

            if (nor_acc && nor_acc->count != n) {
                rollback();
                cgltf_free(data);
                return -1;
            }
            if (col_acc && col_acc->count != n) {
                rollback();
                cgltf_free(data);
                return -1;
            }

            std::vector<float> pos_buf(n * 3);
            cgltf_accessor_unpack_floats(pos, pos_buf.data(), n * 3);

            std::vector<float> nor_buf(n * 3);
            if (nor_acc) {
                cgltf_accessor_unpack_floats(nor_acc, nor_buf.data(), n * 3);
            } else {
                for (cgltf_size i = 0; i < n; ++i) {
                    nor_buf[i * 3 + 0] = 0.f;
                    nor_buf[i * 3 + 1] = 1.f;
                    nor_buf[i * 3 + 2] = 0.f;
                }
            }

            std::vector<float> col_buf(n * 4, 1.0f);
            if (col_acc) {
                if (col_acc->type == cgltf_type_vec3) {
                    std::vector<float> tmp(n * 3);
                    cgltf_accessor_unpack_floats(col_acc, tmp.data(), n * 3);
                    for (cgltf_size i = 0; i < n; ++i) {
                        col_buf[i * 4 + 0] = tmp[i * 3 + 0];
                        col_buf[i * 4 + 1] = tmp[i * 3 + 1];
                        col_buf[i * 4 + 2] = tmp[i * 3 + 2];
                        col_buf[i * 4 + 3] = 1.0f;
                    }
                } else if (col_acc->type == cgltf_type_vec4) {
                    cgltf_accessor_unpack_floats(col_acc, col_buf.data(), n * 4);
                } else {
                    rollback();
                    cgltf_free(data);
                    return -1;
                }
            }

            std::vector<float> verts(n * kPhongFloatsPerVert);
            for (cgltf_size i = 0; i < n; ++i) {
                verts[i * 10 + 0] = pos_buf[i * 3 + 0];
                verts[i * 10 + 1] = pos_buf[i * 3 + 1];
                verts[i * 10 + 2] = pos_buf[i * 3 + 2];
                verts[i * 10 + 3] = nor_buf[i * 3 + 0];
                verts[i * 10 + 4] = nor_buf[i * 3 + 1];
                verts[i * 10 + 5] = nor_buf[i * 3 + 2];
                verts[i * 10 + 6] = col_buf[i * 4 + 0];
                verts[i * 10 + 7] = col_buf[i * 4 + 1];
                verts[i * 10 + 8] = col_buf[i * 4 + 2];
                verts[i * 10 + 9] = col_buf[i * 4 + 3];
            }

            mesh_t mh;
            if (prim.indices) {
                const cgltf_accessor* idx_acc = prim.indices;
                cgltf_size ic = idx_acc->count;
                if (idx_acc->component_type == cgltf_component_type_r_32u) {
                    std::vector<uint32_t> idx(ic);
                    cgltf_accessor_unpack_indices(idx_acc, idx.data(), sizeof(uint32_t), ic);
                    mh = mesh_create(verts.data(), (int32_t)(verts.size() * sizeof(float)), idx.data(),
                                     (int32_t)(ic * sizeof(uint32_t)), BACKEND_INDEX_UINT32);
                } else if (idx_acc->component_type == cgltf_component_type_r_16u) {
                    std::vector<uint16_t> idx(ic);
                    cgltf_accessor_unpack_indices(idx_acc, idx.data(), sizeof(uint16_t), ic);
                    mh = mesh_create(verts.data(), (int32_t)(verts.size() * sizeof(float)), idx.data(),
                                     (int32_t)(ic * sizeof(uint16_t)), BACKEND_INDEX_UINT16);
                } else {
                    rollback();
                    cgltf_free(data);
                    return -1;
                }
            } else {
                if (n > 65535) {
                    std::vector<uint32_t> idx(n);
                    for (cgltf_size i = 0; i < n; ++i) idx[i] = (uint32_t)i;
                    mh = mesh_create(verts.data(), (int32_t)(verts.size() * sizeof(float)), idx.data(),
                                     (int32_t)(n * sizeof(uint32_t)), BACKEND_INDEX_UINT32);
                } else {
                    std::vector<uint16_t> idx(n);
                    for (cgltf_size i = 0; i < n; ++i) idx[i] = (uint16_t)i;
                    mh = mesh_create(verts.data(), (int32_t)(verts.size() * sizeof(float)), idx.data(),
                                     (int32_t)(n * sizeof(uint16_t)), BACKEND_INDEX_UINT16);
                }
            }

            if (mh < 0) {
                rollback();
                cgltf_free(data);
                return -1;
            }

            created.push_back(mh);
            ga.primitives.push_back(mh);
        }
    }

    cgltf_free(data);

    if (ga.primitives.empty()) {
        rollback();
        return -1;
    }

    int32_t aid = (int32_t)gltf_assets.size();
    gltf_assets.push_back(std::move(ga));
    created.clear();
    return aid;
}

void Backend::gltf_unload(int32_t asset) {
    if (asset < 0 || asset >= (int32_t)gltf_assets.size()) return;
    for (mesh_t m : gltf_assets[asset].primitives) {
        mesh_destroy(m);
    }
    gltf_assets[asset].primitives.clear();
}

int32_t Backend::gltf_primitive_count(int32_t asset) const {
    if (asset < 0 || asset >= (int32_t)gltf_assets.size()) return 0;
    return (int32_t)gltf_assets[asset].primitives.size();
}

mesh_t Backend::gltf_primitive_mesh(int32_t asset, int32_t prim) const {
    if (asset < 0 || asset >= (int32_t)gltf_assets.size()) return -1;
    const auto& p = gltf_assets[asset].primitives;
    if (prim < 0 || prim >= (int32_t)p.size()) return -1;
    return p[(size_t)prim];
}
