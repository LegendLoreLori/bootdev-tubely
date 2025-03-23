package main

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/bootdotdev/learn-file-storage-s3-golang-starter/internal/auth"
	"github.com/google/uuid"
)

func (cfg *apiConfig) handlerUploadThumbnail(w http.ResponseWriter, r *http.Request) {
	videoIDString := r.PathValue("videoID")
	videoID, err := uuid.Parse(videoIDString)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid ID", err)
		return
	}

	token, err := auth.GetBearerToken(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't find JWT", err)
		return
	}

	userID, err := auth.ValidateJWT(token, cfg.jwtSecret)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Couldn't validate JWT", err)
		return
	}

	fmt.Println("uploading thumbnail for video", videoID, "by user", userID)

	const maxMemory = 10 << 20
	err = r.ParseMultipartForm(maxMemory)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't parse form", err)
		return
	}

	tn, tnHead, err := r.FormFile("thumbnail")
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get form", err)
		return
	}

	video, err := cfg.db.GetVideo(videoID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't get video", err)
		return
	} else if video.UserID != userID {
		respondWithError(w, http.StatusUnauthorized, "Unauthorized", err)
		return
	}

	mediaType := tnHead.Header.Get("Content-Type")
	ext, _, err := mime.ParseMediaType(mediaType)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Bad mime type provided", err)
		return
	}
	switch ext {
	case "image/png":
	case "image/jpeg":
	default:
		respondWithError(w, http.StatusBadRequest, "Only jpeg or png supported", err)
		return
	} // bad

	s := strings.Split(ext, "/") // i can probably abstract this out later
	ext = s[len(s)-1]
	url := filepath.Join(cfg.assetsRoot, videoIDString+"."+ext)
	f, err := os.Create(url)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't create thumbnail", err)
		return
	}
	_, err = io.Copy(f, tn)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't copy data to thumbnail", err)
		return
	}

	url = fmt.Sprintf("http://localhost:%s/%s", cfg.port, url)
	video.ThumbnailURL = &url
	err = cfg.db.UpdateVideo(video)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Couldn't update video", err)
		return
	}

	respondWithJSON(w, http.StatusOK, video)
}
