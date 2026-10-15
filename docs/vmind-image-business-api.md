# VWorld image v2 forwarding

The adapter uses VMind-Image's minimal multipart business API. Model IDs and Generate RPC envelopes stay unchanged.

| Model / inputs | Upstream path | Text fields | Upload fields |
|---|---|---|---|
| VWorld_wardrobe-1.0, one product image | /v2/images/product_generation | garment_name, garment_role | image |
| VWorld_wardrobe-1.0, parser payload | /v2/images/garment_extraction | garment_name, garment_role, parse_labels | image, parse_mask |
| VWorld_outfit_1.0 | /v2/images/vton | prompt | person_image, garment_image |
| VWorld_bodymodel_1.0 | /v2/images/body_model | gender | image |

A valid active span adds trace_id. Product metadata comes from the existing JSON text part; gender comes from input.model_traits. VTON preserves text-part content and person/garment order. Response decoding remains b64_json → image bytes.

Do not send model, lora_name, lora_strength, size, seed, n, response_format, use_crop, input_mode, or negative_prompt. Do not infer a garment role from inventory_slot or apply role/gender/category validation. Do not enforce crop.applied in this gateway. The model service owns these rules and returns upstream errors before inference. Structural protocol checks (JSON envelope and media mapping) remain here.

Rollout: deploy VMind-Image v2 worker + scheduler/monitor first, then Wardrobe's explicit role mapping, then this ModelHub adapter. Wardrobe's updated payload works with the already merged v1 adapter; v1 remains available for rollback. The existing UserCenter gender envelope needs no change. No deployment is performed by this PR.
