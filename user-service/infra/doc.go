package infra

//go:generate swag init -g infra/doc.go -d . -o docs --parseInternal --ot json,yaml

// Package infra provides configuration, authentication and hashing helpers for
// the user-service. The OpenAPI contract under docs/ is GENERATED from the
// handler annotations and must not be edited by hand (it is gitignored).
