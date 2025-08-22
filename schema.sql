CREATE TABLE files (
    id          INTEGER PRIMARY KEY,
    name    VARCHAR(255),
    description VARCHAR(255),
    type VARCHAR(255),
    file_path VARCHAR(255)
);

CREATE TABLE files_tags (
    file_id INTEGER,
    tag_id INTEGER,
    FOREIGN KEY (file_id) REFERENCES files(id),
    FOREIGN KEY (tag_id) REFERENCES tags(id)
);

CREATE TABLE tags (
    id INTEGER PRIMARY KEY,
    name VARCHAR(255)
);