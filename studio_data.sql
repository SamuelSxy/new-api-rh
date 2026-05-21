--
-- PostgreSQL database dump
--

\restrict QXYpfLJKc3Xb3kBnY482hqZfX5MrBc52b89hevIi4XrleUHq6lax0lO7GugqLLA

-- Dumped from database version 15.17 (Debian 15.17-1.pgdg13+1)
-- Dumped by pg_dump version 15.17 (Debian 15.17-1.pgdg13+1)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Data for Name: studio_form_schemas; Type: TABLE DATA; Schema: public; Owner: root
--

INSERT INTO public.studio_form_schemas VALUES (3, 'Happyhorse Video Form', 'video', 'happyhorse', 1, '{
  "name": "Happyhorse Video Form",
  "modelType": "video",
  "modelName": "happyhorse",
  "fields": [
    {
      "key": "resolution",
      "label": "Resolution",
      "type": "select",
      "required": true,
      "options": [
        {
          "label": "480p",
          "value": "480"
        },
        {
          "label": "720p",
          "value": "720"
        },
        {
          "label": "1080p",
          "value": "1080"
        }
      ],
      "defaultValue": "720"
    },
    {
      "key": "image_urls",
      "label": "Images (Local upload, multiple)",
      "type": "image_upload",
      "required": true,
      "max": 8,
      "helpText": "Images will be converted to base64 data URLs automatically"
    },
    {
      "key": "duration",
      "label": "Duration (s)",
      "type": "number",
      "required": true,
      "min": 3,
      "max": 15,
      "step": 1,
      "defaultValue": 6
    }
  ]
}', 'Video generation form for happyhorse', 1, 1778395179, 1778693791, NULL);
INSERT INTO public.studio_form_schemas VALUES (4, 'doubao-seedance-2-0-fast-260128 Video Form', 'video', 'doubao-seedance-2-0-fast-260128', 1, '{
  "name": "Seedance 2.0 Fast Video Form",
  "modelType": "video",
  "modelName": "seedance2.0-fast",
  "fields": [
    {
      "key": "resolution",
      "label": "Resolution",
      "type": "select",
      "required": true,
      "options": [
        {
          "label": "480p",
          "value": "480"
        },
        {
          "label": "720p",
          "value": "720"
        }
      ],
      "defaultValue": "720"
    },
    {
      "key": "image_urls",
      "label": "Images (Local upload, multiple)",
      "type": "image_upload",
      "required": true,
      "max": 10,
      "helpText": "Images will be converted to base64 data URLs automatically"
    },
    {
      "key": "video_url",
      "label": "Reference Video URL",
      "type": "text",
      "placeholder": "https://example.com/reference.mp4"
    },
    {
      "key": "duration",
      "label": "Duration (s)",
      "type": "number",
      "required": true,
      "min": 4,
      "max": 15,
      "step": 1,
      "defaultValue": 8
    }
  ]
}', 'Video generation form for doubao-seedance-2-0-fast-260128', 1, 1778402176, 1778402176, NULL);
INSERT INTO public.studio_form_schemas VALUES (5, 'rhart-image-n-pro-official/edit Image Form', 'image', 'rhart-image-n-pro-official/edit', 1, '{
  "name": "RunningHub Image Edit Form",
  "modelType": "image",
  "modelName": "rhart-image-n-pro-official/edit",
  "fields": [
    {
      "key": "image_urls",
      "label": "Reference Images",
      "type": "image_upload",
      "helpText": "Upload one or more reference images"
    },
    {
      "key": "aspect_ratio",
      "label": "Aspect Ratio",
      "type": "select",
      "options": [
        {
          "label": "1:1",
          "value": "1:1"
        },
        {
          "label": "4:3",
          "value": "4:3"
        },
        {
          "label": "3:4",
          "value": "3:4"
        },
        {
          "label": "16:9",
          "value": "16:9"
        },
        {
          "label": "9:16",
          "value": "9:16"
        }
      ],
      "defaultValue": "9:16"
    },
    {
      "key": "resolution",
      "label": "Resolution",
      "type": "select",
      "options": [
        {
          "label": "1k",
          "value": "1k"
        },
        {
          "label": "2k",
          "value": "2k"
        },
        {
          "label": "4k",
          "value": "4k"
        }
      ],
      "defaultValue": "1k"
    }
  ]
}', 'Image generation form for rhart-image-n-pro-official/edit', 1, 1778609206, 1778609218, NULL);
INSERT INTO public.studio_form_schemas VALUES (1, 'Seedance 2.0 Fast Video Form', 'video', 'seedance2.0-fast', 1, '{
  "name": "Seedance 2.0 Fast Video Form",
  "modelType": "video",
  "modelName": "seedance2.0-fast",
  "fields": [
    {
      "key": "resolution",
      "label": "Resolution",
      "type": "select",
      "required": true,
      "options": [
        {
          "label": "480p",
          "value": "480"
        },
        {
          "label": "720p",
          "value": "720"
        }
      ],
      "defaultValue": "720"
    },
    {
      "key": "image_urls",
      "label": "Images (Local upload, multiple)",
      "type": "image_upload",
      "required": true,
      "max": 10,
      "helpText": "Images will be converted to base64 data URLs automatically"
    },
    {
      "key": "video_url",
      "label": "Reference Video URL",
      "type": "text",
      "placeholder": "https://example.com/reference.mp4"
    },
    {
      "key": "duration",
      "label": "Duration (s)",
      "type": "number",
      "required": true,
      "min": 4,
      "max": 15,
      "step": 1,
      "defaultValue": 8
    }
  ]
}', 'Video generation form for seedance2.0-fast', 1, 1778395179, 1778693791, NULL);
INSERT INTO public.studio_form_schemas VALUES (6, 'rhart-audio/text-to-audio/speech-2.8-turbo Voice Form', 'voice', 'rhart-audio/text-to-audio/speech-2.8-turbo', 1, '{
  "name": "RunningHub Speech 2.8 Turbo Form",
  "modelType": "voice",
  "modelName": "rhart-audio/text-to-audio/speech-2.8-turbo",
  "fields": [
    {
      "key": "voice",
      "label": "Voice",
      "type": "text",
      "placeholder": "Elegant_Man",
      "defaultValue": "Elegant_Man"
    },
    {
      "key": "speed",
      "label": "Speed",
      "type": "number",
      "min": 0.25,
      "max": 4,
      "step": 0.05,
      "defaultValue": 1
    },
    {
      "key": "volume",
      "label": "Volume",
      "type": "number",
      "min": 0,
      "max": 2,
      "step": 0.1,
      "defaultValue": 1
    },
    {
      "key": "pitch",
      "label": "Pitch",
      "type": "number",
      "min": -12,
      "max": 12,
      "step": 1,
      "defaultValue": 0
    },
    {
      "key": "emotion",
      "label": "Emotion",
      "type": "text",
      "placeholder": "happy",
      "defaultValue": "happy"
    },
    {
      "key": "response_format",
      "label": "Response Format",
      "type": "select",
      "options": [
        {
          "label": "mp3",
          "value": "mp3"
        },
        {
          "label": "wav",
          "value": "wav"
        },
        {
          "label": "pcm",
          "value": "pcm"
        }
      ],
      "defaultValue": "mp3"
    },
    {
      "key": "enable_base64_output",
      "label": "Enable Base64 Output",
      "type": "switch",
      "defaultValue": false
    },
    {
      "key": "english_normalization",
      "label": "English Normalization",
      "type": "switch",
      "defaultValue": false
    }
  ]
}', 'Voice generation form for rhart-audio/text-to-audio/speech-2.8-turbo', 1, 1778609236, 1778609240, NULL);
INSERT INTO public.studio_form_schemas VALUES (2, 'Seedance 2.0 Video Form', 'video', 'seedance2.0', 1, '{
  "name": "Seedance 2.0 Video Form",
  "modelType": "video",
  "modelName": "seedance2.0",
  "fields": [
    {
      "key": "resolution",
      "label": "Resolution",
      "type": "select",
      "required": true,
      "options": [
        {
          "label": "480p",
          "value": "480"
        },
        {
          "label": "720p",
          "value": "720"
        },
        {
          "label": "1080p",
          "value": "1080"
        }
      ],
      "defaultValue": "1080"
    },
    {
      "key": "image_urls",
      "label": "Images (Local upload, multiple)",
      "type": "image_upload",
      "required": true,
      "max": 10,
      "helpText": "Images will be converted to base64 data URLs automatically"
    },
    {
      "key": "video_url",
      "label": "Reference Video URL",
      "type": "text",
      "placeholder": "https://example.com/reference.mp4"
    },
    {
      "key": "duration",
      "label": "Duration (s)",
      "type": "number",
      "required": true,
      "min": 4,
      "max": 15,
      "step": 1,
      "defaultValue": 8
    }
  ]
}', 'Video generation form for seedance2.0', 1, 1778395179, 1778693791, NULL);
INSERT INTO public.studio_form_schemas VALUES (8, 'alibaba/happyhorse-1.0/reference-to-video Video Form', 'video', 'alibaba/happyhorse-1.0/reference-to-video', 1, '{
  "name": "Happyhorse Video Form",
  "modelType": "video",
  "modelName": "happyhorse",
  "fields": [
    {
      "key": "resolution",
      "label": "Resolution",
      "type": "select",
      "required": true,
      "options": [
        {
          "label": "720p",
          "value": "720p"
        },
        {
          "label": "1080p",
          "value": "1080p"
        }
      ],
      "defaultValue": "720p"
    },
    {
      "key": "imageUrls",
      "label": "Images (Local upload, multiple)",
      "type": "image_upload",
      "required": true,
      "max": 8,
      "helpText": "Images will be converted to base64 data URLs automatically"
    },
    {
      "key": "duration",
      "label": "Duration (s)",
      "type": "number",
      "required": true,
      "min": 3,
      "max": 15,
      "step": 1,
      "defaultValue": 6
    }
  ]
}', 'Video generation form for alibaba/happyhorse-1.0/reference-to-video', 1, 1778694013, 1778694270, NULL);
INSERT INTO public.studio_form_schemas VALUES (7, 'doubao-seedance-2-0-260128 Video Form', 'video', 'doubao-seedance-2-0-260128', 1, '{
  "name": "Seedance 2.0 Video Form",
  "modelType": "video",
  "modelName": "seedance2.0",
  "fields": [
    {
      "key": "resolution",
      "label": "Resolution",
      "type": "select",
      "required": true,
      "options": [
        {
          "label": "480p",
          "value": "480"
        },
        {
          "label": "720p",
          "value": "720"
        },
        {
          "label": "1080p",
          "value": "1080"
        }
      ],
      "defaultValue": "1080"
    },
    {
      "key": "image_urls",
      "label": "Images (Local upload, multiple)",
      "type": "image_upload",
      "required": true,
      "max": 10,
      "helpText": "Images will be converted to base64 data URLs automatically"
    },
    {
      "key": "video_url",
      "label": "Reference Video URL",
      "type": "text",
      "placeholder": "https://example.com/reference.mp4"
    },
    {
      "key": "duration",
      "label": "Duration (s)",
      "type": "number",
      "required": true,
      "min": 4,
      "max": 15,
      "step": 1,
      "defaultValue": 8
    }
  ]
}', 'Video generation form for doubao-seedance-2-0-260128', 1, 1778693946, 1778693946, NULL);


--
-- Data for Name: studio_model_configs; Type: TABLE DATA; Schema: public; Owner: root
--

INSERT INTO public.studio_model_configs VALUES (4, 'testbysamuel', 'alibaba/happyhorse-1.0/reference-to-video', 'script', '', '', '', '', 1, 1778400247, 1778400247, '2026-05-10 08:09:36.277331+00');
INSERT INTO public.studio_model_configs VALUES (5, 'RunningHub Image Edit', 'rhart-image-n-pro-official/edit', 'image', 'RunningHub image-to-image editing workflow', '', '', '', 1, 1778609206, 1778609214, NULL);
INSERT INTO public.studio_model_configs VALUES (6, 'RunningHub Speech 2.8 Turbo', 'rhart-audio/text-to-audio/speech-2.8-turbo', 'voice', 'RunningHub text-to-audio speech workflow', '', '', '', 1, 1778609236, 1778609239, NULL);
INSERT INTO public.studio_model_configs VALUES (2, 'Seedance 2.0', 'seedance2.0', 'video', 'High quality video generation', '', '', '', 1, 1778395179, 1778693448, '2026-05-13 17:32:04.56673+00');
INSERT INTO public.studio_model_configs VALUES (7, 'Seedance 2.0 Fast', 'seedance2.0-fast', 'video', 'Fast video generation', '', '', '', 1, 1778693443, 1778693448, '2026-05-13 17:32:16.456919+00');
INSERT INTO public.studio_model_configs VALUES (10, 'Happyhorse', 'happyhorse', 'video', 'Creative short video generation', '', '', '', 1, 1778693596, 1778693596, '2026-05-13 17:33:45.277748+00');
INSERT INTO public.studio_model_configs VALUES (8, 'Seedance 2.0 Fast', 'seedance2.0-fast', 'video', 'Fast video generation', '', '', '', 1, 1778693596, 1778693596, '2026-05-13 17:33:53.249084+00');
INSERT INTO public.studio_model_configs VALUES (9, 'Seedance 2.0', 'doubao-seedance-2-0-260128', 'video', 'High quality video generation', '', '', '', 1, 1778693596, 1778693756, '2026-05-13 17:36:53.709121+00');
INSERT INTO public.studio_model_configs VALUES (3, 'Happyhorse', 'alibaba/happyhorse-1.0/reference-to-video', 'video', 'Creative short video generation', '', '', '', 1, 1778395179, 1778693593, '2026-05-13 17:37:09.218058+00');
INSERT INTO public.studio_model_configs VALUES (1, 'Seedance 2.0 Fast', 'doubao-seedance-2-0-fast-260128', 'video', 'Fast video generation', '', '', '', 1, 1778395179, 1778402168, '2026-05-13 17:37:11.97104+00');
INSERT INTO public.studio_model_configs VALUES (11, 'Seedance 2.0 Fast', 'doubao-seedance-2-0-fast-260128', 'video', 'Fast video generation', '', '', '', 1, 1778693791, 1778693887, NULL);
INSERT INTO public.studio_model_configs VALUES (12, 'Seedance 2.0', 'doubao-seedance-2-0-260128', 'video', 'High quality video generation', '', '', '', 1, 1778693791, 1778693944, NULL);
INSERT INTO public.studio_model_configs VALUES (13, 'Happyhorse', 'alibaba/happyhorse-1.0/reference-to-video', 'video', 'Creative short video generation', '', '', '', 1, 1778693791, 1778694011, NULL);


--
-- Name: studio_form_schemas_id_seq; Type: SEQUENCE SET; Schema: public; Owner: root
--

SELECT pg_catalog.setval('public.studio_form_schemas_id_seq', 8, true);


--
-- Name: studio_model_configs_id_seq; Type: SEQUENCE SET; Schema: public; Owner: root
--

SELECT pg_catalog.setval('public.studio_model_configs_id_seq', 13, true);


--
-- PostgreSQL database dump complete
--

\unrestrict QXYpfLJKc3Xb3kBnY482hqZfX5MrBc52b89hevIi4XrleUHq6lax0lO7GugqLLA

