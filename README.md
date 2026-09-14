---
title: Partipix
---

## Basic Idea

At events, photos get scattered across everyone's phones. Partipix centralizes them: guests join an event via a share code, upload their photos, and the platform automatically surfaces every photo each person appears in using facial recognition.

## Architecture

The backend is a Go service using `fasthttp` for HTTP handling and SQLite via GORM for persistence. The frontend is a SvelteKit app. When a photo is uploaded, it is passed synchronously through the facial recognition pipeline before the API response is returned—so the client receives face match data alongside the freshly saved photo.

```
Upload Photo
     │
     ▼
Extract faces (dlib via go-face)
     │
     ▼
Compute 128-dim descriptor per face
     │
     ▼
Compare against stored embeddings (event participants only)
     │
     ▼
Persist: PhotoPersonMatch + FaceLocation
     │
     ▼
Return Photo + []FaceMatch to client
```

## Facial Recognition Pipeline

Face detection and embedding are both handled by `github.com/Kagami/go-face`, which wraps dlib's ResNet-based face recognition model (`dlib_face_recognition_resnet_model_v1.dat`). Landmark detection uses `shape_predictor_5_face_landmarks.dat` and the face detector uses `mmod_human_face_detector.dat`.

**Reference photos** — each user uploads one or more profile photos. On upload, the backend detects the face, extracts its 128-float32 descriptor, serializes it to 512 bytes (`float32` × 128), and stores it in the `face_embeddings` table. This pre-computation means matching at photo-upload time is a pure in-memory vector comparison loop rather than running the full recognition model again.

**Matching** — for each face detected in an event photo, the backend:

1. Computes the face descriptor from the cropped region (padded 20% on each side, resized to 150×150 with Lanczos).
2. Loads stored embeddings only for participants in that event.
3. Computes squared Euclidean distance between the query descriptor and each stored embedding; similarity = `1.0 − distance`.
4. Records a match if similarity ≥ 0.5.

The same matching logic runs retroactively when a new participant joins an event—existing photos are re-scanned against the newcomer's embeddings.

## Storage

Uploaded photos are organized on disk under `uploads/<eventId>/`. Profile photos (used as reference images) live under `uploads/pfps/`. The HTTP server serves these files directly, restricting extensions to `.jpg`, `.jpeg`, `.png`, and `.webp`.

## Frontend

Built with SvelteKit and styled with Tailwind CSS. Key routes:

- `/events` — list events the user has created or joined
- `/events/[id]` — event detail: guest list with upload status, photo grid with face-match overlays
- `/profile` — manage reference photos used for face matching
- `/admin` — user and event management (admin only)

Face match bounding boxes are rendered as overlays on each photo, tagged with the matched user's name and confidence score.

<img>
