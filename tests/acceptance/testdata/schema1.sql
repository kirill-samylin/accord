-- Canonical schema1 DDL from dc9c74c internal/store/store.go. This is a
-- synthetic fixture, not a database claimed to have run under an old binary.
CREATE TABLE metadata (id INTEGER PRIMARY KEY CHECK(id=1),sequence INTEGER NOT NULL);
INSERT INTO metadata VALUES(1,2);
CREATE TABLE tasks (key TEXT PRIMARY KEY,data TEXT NOT NULL);
CREATE TABLE sessions (key TEXT PRIMARY KEY,data TEXT NOT NULL);
CREATE TABLE role_generations (role TEXT PRIMARY KEY,generation INTEGER NOT NULL);
CREATE TABLE events (cursor INTEGER PRIMARY KEY AUTOINCREMENT,data TEXT NOT NULL);
CREATE TABLE idempotency (key TEXT PRIMARY KEY,hash TEXT NOT NULL,response TEXT NOT NULL);
CREATE TABLE tokens (id TEXT PRIMARY KEY,hash TEXT UNIQUE NOT NULL,owner TEXT NOT NULL,machine TEXT NOT NULL,expires INTEGER NOT NULL,revoked INTEGER NOT NULL DEFAULT 0);
PRAGMA user_version=1;
INSERT INTO tasks VALUES('E-1','{"key":"E-1","type":"epic","title":"Legacy synthetic epic","description":"Canonical fixture","role":"","state":"open","owner":"legacy-owner","ready_at":"2026-10-18T00:00:00Z","version":1,"generation":0,"evidence":{}}');
INSERT INTO tasks VALUES('T-2','{"key":"T-2","type":"task","epic_key":"E-1","title":"Legacy synthetic task","description":"No invented timestamps","role":"developer","state":"done","ready_at":"2026-10-18T00:00:00Z","version":1,"generation":0,"evidence":{}}');
INSERT INTO events(data) VALUES('{"cursor":1,"type":"task_done","task_key":"T-2","at":"2026-10-18T00:00:00Z"}');
INSERT INTO idempotency VALUES('synthetic-legacy-replay','synthetic-hash','{"legacy":"exact response bytes"}');
