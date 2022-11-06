package p2p

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type MyError struct {
	err error
}

func (e MyError) Error() string {
	return e.err.Error()
}

type apiFunc func(w http.ResponseWriter, r *http.Request) error

func makeHTTPHandleFunc(f apiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := f(w, r); err != nil {
			JSON(w, http.StatusBadRequest, map[string]any{"error": err.Error()})
		}
	}
}

func JSON(w http.ResponseWriter, status int, v any) error {
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

type APIServer struct {
	listenAddr string
	game       *Game
}

func NewAPIServer(listenAddr string, game *Game) *APIServer {
	return &APIServer{
		game:       game,
		listenAddr: listenAddr,
	}
}

func (s *APIServer) Run() {
	r := mux.NewRouter()

	r.HandleFunc("/ready", makeHTTPHandleFunc(s.handlePlayerReady))
	r.HandleFunc("/fold", makeHTTPHandleFunc(s.handlePlayerFold))

	http.ListenAndServe(s.listenAddr, r)
}

func (s *APIServer) handlePlayerFold(w http.ResponseWriter, r *http.Request) error {
	s.game.Fold()
	return JSON(w, http.StatusOK, []byte("FOLDED"))
}

func (s *APIServer) handlePlayerReady(w http.ResponseWriter, r *http.Request) error {
	s.game.SetReady()
	return JSON(w, http.StatusOK, []byte("READY"))
}
