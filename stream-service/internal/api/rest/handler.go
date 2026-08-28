package rest

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"

	"radio/stream-service/internal/api/rest/handlers/dto"
	"radio/stream-service/internal/application"
	"radio/stream-service/internal/domain/models"
)

type Handler struct {
	svc *application.Service
}

func (h *Handler) healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

func NewHandler(svc *application.Service) http.Handler {
	h := &Handler{svc: svc}

	mux := http.NewServeMux()

	// Health
	mux.HandleFunc("GET /healthz", h.healthz)

	// Stream
	mux.HandleFunc("GET /streams", h.getUserStream)
	mux.HandleFunc("GET /streams/", h.getUserStream)
	mux.HandleFunc("GET /streams/feed", h.feed)
	mux.HandleFunc("GET /streams/{id}", h.getStream)
	mux.HandleFunc("PUT /streams/{id}", h.updateStream)
	mux.HandleFunc("DELETE /streams/{id}", h.deleteStream)

	// Playback control
	mux.HandleFunc("POST /streams/{id}/start", h.startStream)
	mux.HandleFunc("POST /streams/{id}/stop", h.stopStream)

	// Stream status
	mux.HandleFunc("GET /streams/{id}/state", h.getState)

	// Queue
	mux.HandleFunc("GET /streams/{id}/queue", h.listQueue)
	mux.HandleFunc("POST /streams/{id}/queue", h.addToQueue)
	mux.HandleFunc("DELETE /streams/{id}/queue/{itemId}", h.removeFromQueue)
	mux.HandleFunc("PUT /streams/{id}/queue/{itemId}", h.moveQueueItem)

	// Hashtags
	mux.HandleFunc("GET /streams/{id}/hashtags", h.listHashtags)
	mux.HandleFunc("POST /streams/{id}/hashtags", h.addHashtag)
	mux.HandleFunc("DELETE /streams/{id}/hashtags/{hashtagId}", h.removeHashtag)

	return mux
}

func (h *Handler) ensureStream(w http.ResponseWriter, r *http.Request, ownerID uuid.UUID) (*models.Stream, bool) {
	stream, err := h.svc.GetOrCreateStream(r.Context(), ownerID)
	if err != nil {
		WriteServiceError(w, err)
		return nil, false
	}
	return stream, true
}

func (h *Handler) getUserStream(w http.ResponseWriter, r *http.Request) {
	ownerID, ok := ownerOrError(w, r)
	if !ok {
		return
	}

	stream, ok := h.ensureStream(w, r, ownerID)
	if !ok {
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

func (h *Handler) feed(w http.ResponseWriter, r *http.Request) {
	streamIDs, err := h.svc.ListActiveStreams(r.Context())
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	result := make([]dto.StreamResponse, 0, len(streamIDs))
	for _, streamID := range streamIDs {
		stream, err := h.svc.GetStream(r.Context(), streamID)
		if err != nil {
			continue
		}
		result = append(result, dto.StreamResponse{
			ID:          stream.ID,
			Name:        stream.Name,
			Description: stream.Description,
			Loop:        stream.Loop,
			CreatedAt:   stream.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
			UpdatedAt:   stream.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	WriteJSON(w, http.StatusOK, result)
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

	if _, ok := h.ensureStream(w, r, ownerID); !ok {
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

	if _, ok := h.ensureStream(w, r, ownerID); !ok {
		return
	}

	if err := h.svc.Start(r.Context(), id); err != nil {
		WriteServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

	if err := h.svc.Stop(r.Context(), id); err != nil {
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

	st, err := h.svc.GetStatus(r.Context(), id)
	if err != nil {
		WriteServiceError(w, err)
		return
	}

	WriteJSON(w, http.StatusOK, dto.StreamStatusResponse{
		IsActive:      st.IsActive,
		CurrentItemID: st.CurrentItemID,
		CurrentSongID: st.CurrentSongID,
		Position:      st.Position,
		QueueLength:   st.QueueLength,
	})
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
			ID:       item.ID,
			SongID:   item.SongID,
			Position: item.Position,
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

	if _, ok := h.ensureStream(w, r, ownerID); !ok {
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

	WriteJSON(w, http.StatusCreated, dto.QueueItemResponse{
		ID:       item.ID,
		SongID:   item.SongID,
		Position: item.Position,
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

	if _, ok := h.ensureStream(w, r, ownerID); !ok {
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

	if _, ok := h.ensureStream(w, r, ownerID); !ok {
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
	for i, hashtag := range hashtagList {
		resp[i] = dto.HashtagResponse{
			ID:        hashtag.ID,
			StreamID:  hashtag.StreamID,
			Name:      hashtag.Name,
			CreatedAt: hashtag.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
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

	if _, ok := h.ensureStream(w, r, ownerID); !ok {
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

	if _, ok := h.ensureStream(w, r, ownerID); !ok {
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
