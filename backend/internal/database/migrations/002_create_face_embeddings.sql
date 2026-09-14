CREATE TABLE IF NOT EXISTS face_embeddings (
    id TEXT PRIMARY KEY,
    reference_photo_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    embedding_vector BLOB NOT NULL,
    created_at TIMESTAMP NOT NULL,
    FOREIGN KEY(reference_photo_id) REFERENCES reference_photos(id) ON DELETE CASCADE,
    FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_face_embeddings_user_id ON face_embeddings(user_id);
CREATE INDEX IF NOT EXISTS idx_face_embeddings_reference_photo_id ON face_embeddings(reference_photo_id);
