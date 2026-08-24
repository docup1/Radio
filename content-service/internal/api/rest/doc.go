package rest

//go:generate swag init -g internal/api/rest/doc.go -o docs --parseInternal

// Package rest implements the public user-facing REST API for the content
// service (songs, melodies, images, playlists and chunked uploads). The
// OpenAPI contract under docs/ is GENERATED from the handler annotations and
// must not be edited by hand (it is gitignored).
