package rest

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"radio/stream-service/internal/api/rest/handlers/dto"
	"radio/stream-service/internal/application"
)

type Handler struct {
	svc    *application.Service
	worker *application.Worker
}

func (h *Handler) healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func NewHandler(svc *application.Service, worker *application.Worker) http.Handler {
	h := &Handler{svc: svc, worker: worker}

	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /healthz", h.healthz)

	// Stream CRUD
	mux.HandleFunc("POST /streams", h.createStream)
	mux.HandleFunc("GET /streams/{id}", h.getStream)
	mux.HandleFunc("PUT /streams/{id}", h.updateStream)
	mux.HandleFunc("DELETE /streams/{id}", h.deleteStream)

	// Stream state
	mux.HandleFunc("GET /streams/{id}/state", h.getState)
	mux.HandleFunc("POST /streams/{id}/start", h.startStream)
	mux.HandleFunc("POST /streams/{id}/stop", h.stopStream)

	// Queue
	mux.HandleFunc("GET /streams/{id}/queue", h.listQueue)
	mux.HandleFunc("POST /streams/{id}/queue", h.addToQueue)
	mux.HandleFunc("DELETE /streams/{id}/queue/{itemId}", h.removeFromQueue)
	mux.HandleFunc("PUT /streams/{id}/queue/{itemId}", h.moveQueueItem)

	// Hashtags
	mux.HandleFunc("GET /streams/{id}/hashtags", h.listHashtags)
	mux.HandleFunc("POST /streams/{id}/hashtags", h.addHashtag)
	mux.HandleFunc("DELETE /streams/{id}/hashtags/{hashtagId}", h.removeHashtag)

	// Internal (for sender)
	mux.HandleFunc("POST /internal/streams/{id}/advance", h.internalAdvance)

	return mux
}

func (h *Handler) createStream(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := ownerOrError(w, r)
	if !ok {
		return
	}

	var req dto.CreateStreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	if req.Name == "" {
		WriteError(w, http.StatusBadRequest, "name is required")
		return
	}

	stream, err := h.svc.CreateStream(r.Context(), ownerID, req.Name, req.Description, req.Loop)
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, dto.StreamResponse{
		ID:          stream.ID,
		Name:        stream.Name,
		Description: stream.Description,
		Loop:        stream.Loop,
		CreatedAt:   stream.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   stream.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

func (h *Handler) getStream(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	stream, err := h.svc.GetStream(r.Context(), id)
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, dto.StreamResponse{
		ID:          stream.ID,
		Name:        stream.Name,
		Description: stream.Description,
		Loop:        stream.Loop,
		CreatedAt:   stream.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   stream.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

func (h *Handler) updateStream(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := ownerOrError(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if id != ownerID {
		WriteError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req dto.UpdateStreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}

	stream, err := h.svc.UpdateStream(r.Context(), id, req.Name, req.Description, req.Loop)
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, dto.StreamResponse{
		ID:          stream.ID,
		Name:        stream.Name,
		Description: stream.Description,
		Loop:        stream.Loop,
		CreatedAt:   stream.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   stream.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

func (h *Handler) deleteStream(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := ownerOrError(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if id != ownerID {
		WriteError(w, http.StatusForbidden, "forbidden")
		return
	}

	if err := h.svc.DeleteStream(r.Context(), id); err != nil {
		WriteServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) getState(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	st, err := h.svc.GetState(r.Context(), id)
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	resp := dto.StreamStateResponse{
		StreamID:       st.StreamID,
		CurrentQueueID: st.CurrentQueueID,
		IsActive:       st.IsActive,
		Revision:       st.Revision,
		UpdatedAt:      st.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if st.StartedAt != nil {
		s := st.StartedAt.Format("2006-01-02T15:04:05Z07:00")
		resp.StartedAt = &s
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) startStream(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := ownerOrError(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if id != ownerID {
		WriteError(w, http.StatusForbidden, "forbidden")
		return
	}

	st, err := h.svc.StartStream(r.Context(), id)
	if err != nil {
		WriteServiceError(w, err)
		return
	}
	h.worker.Signal()

	resp := dto.StreamStateResponse{
		StreamID:       st.StreamID,
		CurrentQueueID: st.CurrentQueueID,
		IsActive:       st.IsActive,
		Revision:       st.Revision,
		UpdatedAt:      st.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if st.StartedAt != nil {
		s := st.StartedAt.Format("2006-01-02T15:04:05Z07:00")
		resp.StartedAt = &s
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) stopStream(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := ownerOrError(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if id != ownerID {
		WriteError(w, http.StatusForbidden, "forbidden")
		return
	}

	st, err := h.svc.StopStream(r.Context(), id)
	if err != nil {
		WriteServiceError(w, err)
		return
	}
	h.worker.Signal()

	resp := dto.StreamStateResponse{
		StreamID:       st.StreamID,
		CurrentQueueID: st.CurrentQueueID,
		IsActive:       st.IsActive,
		Revision:       st.Revision,
		UpdatedAt:      st.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) listQueue(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	items, err := h.svc.ListQueue(r.Context(), id)
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	resp := make([]dto.QueueItemResponse, len(items))
	for i, item := range items {
		resp[i] = dto.QueueItemResponse{
			ID:        item.ID,
			StreamID:  item.StreamID,
			SongID:    item.SongID,
			Position:  item.Position,
			CreatedAt: item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) addToQueue(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := ownerOrError(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if id != ownerID {
		WriteError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req dto.AddToQueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}

	item, err := h.svc.AddToQueue(r.Context(), id, req.SongID)
	if err != nil {
		WriteServiceError(w, err)
		return
	}
	h.worker.Signal()

	WriteJSON(w, http.StatusCreated, dto.QueueItemResponse{
		ID:        item.ID,
		StreamID:  item.StreamID,
		SongID:    item.SongID,
		Position:  item.Position,
		CreatedAt: item.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

func (h *Handler) removeFromQueue(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := ownerOrError(w, r)
	if !ok {
		return
	}
	streamID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if streamID != ownerID {
		WriteError(w, http.StatusForbidden, "forbidden")
		return
	}
	itemID, err := uuid.Parse(r.PathValue("itemId"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	if err := h.svc.RemoveFromQueue(r.Context(), streamID, itemID); err != nil {
		WriteServiceError(w, err)
		return
	}
	h.worker.Signal()

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) moveQueueItem(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := ownerOrError(w, r)
	if !ok {
		return
	}
	streamID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if streamID != ownerID {
		WriteError(w, http.StatusForbidden, "forbidden")
		return
	}
	itemID, err := uuid.Parse(r.PathValue("itemId"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid item id")
		return
	}

	var req dto.MoveQueueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}

	if err := h.svc.MoveQueueItem(r.Context(), streamID, itemID, req.Position); err != nil {
		WriteServiceError(w, err)
		return
	}
	h.worker.Signal()

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) listHashtags(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	hashtagList, err := h.svc.ListHashtags(r.Context(), id)
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	resp := make([]dto.HashtagResponse, len(hashtagList))
	for i, h := range hashtagList {
		resp[i] = dto.HashtagResponse{
			ID:        h.ID,
			StreamID:  h.StreamID,
			Name:      h.Name,
			CreatedAt: h.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		}
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *Handler) addHashtag(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := ownerOrError(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if id != ownerID {
		WriteError(w, http.StatusForbidden, "forbidden")
		return
	}

	var req dto.AddHashtagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}

	hashtag, err := h.svc.AddHashtag(r.Context(), id, req.Name)
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusCreated, dto.HashtagResponse{
		ID:        hashtag.ID,
		StreamID:  hashtag.StreamID,
		Name:      hashtag.Name,
		CreatedAt: hashtag.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

func (h *Handler) removeHashtag(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := ownerOrError(w, r)
	if !ok {
		return
	}
	streamID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}
	if streamID != ownerID {
		WriteError(w, http.StatusForbidden, "forbidden")
		return
	}
	hashtagID, err := uuid.Parse(r.PathValue("hashtagId"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid hashtag id")
		return
	}

	if err := h.svc.RemoveHashtag(r.Context(), streamID, hashtagID); err != nil {
		WriteServiceError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) internalAdvance(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		WriteError(w, http.StatusBadRequest, "invalid id")
		return
	}

	st, err := h.svc.AdvanceSong(r.Context(), id)
	if err != nil {
		WriteServiceError(w, err)
		return
	}
	h.worker.Signal()

	resp := dto.StreamStateResponse{
		StreamID:       st.StreamID,
		CurrentQueueID: st.CurrentQueueID,
		IsActive:       st.IsActive,
		Revision:       st.Revision,
		UpdatedAt:      st.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
	if st.StartedAt != nil {
		s := st.StartedAt.Format("2006-01-02T15:04:05Z07:00")
		resp.StartedAt = &s
	}
	WriteJSON(w, http.StatusOK, resp)
}
