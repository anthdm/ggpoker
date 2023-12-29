package p2p

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

const (
	RouteReady = "/ready"
	RouteFold  = "/fold"
	RouteCheck = "/check"
	RouteBet   = "/bet/{value}"
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
	game       *GameState
}

func NewAPIServer(listenAddr string, game *GameState) *APIServer {
	return &APIServer{
		game:       game,
		listenAddr: listenAddr,
	}
}

func (s *APIServer) Run() {
	r := mux.NewRouter()

	r.HandleFunc(RouteReady, makeHTTPHandleFunc(s.handlePlayerReady))
	r.HandleFunc(RouteFold, makeHTTPHandleFunc(s.handlePlayerFold))
	r.HandleFunc(RouteCheck, makeHTTPHandleFunc(s.handlePlayerCheck))
	r.HandleFunc(RouteBet, makeHTTPHandleFunc(s.handlePlayerBet))

	http.ListenAndServe(s.listenAddr, r)
}

func (s *APIServer) handlePlayerBet(w http.ResponseWriter, r *http.Request) error {
	valueStr := mux.Vars(r)["value"]
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		return fmt.Errorf("error converting value to int: %w", err)
	}

	if err := s.game.TakeAction(PlayerActionBet, value); err != nil {
		return fmt.Errorf("error taking bet action: %w", err)
	}

	return JSON(w, http.StatusOK, fmt.Sprintf("value:%d", value))
}

func (s *APIServer) handlePlayerCheck(w http.ResponseWriter, r *http.Request) error {
	if err := s.game.TakeAction(PlayerActionCheck, 0); err != nil {
		return err
	}
	return JSON(w, http.StatusOK, "CHECKED")
}

func (s *APIServer) handlePlayerFold(w http.ResponseWriter, r *http.Request) error {
	if err := s.game.TakeAction(PlayerActionFold, 0); err != nil {
		return err
	}
	return JSON(w, http.StatusOK, "FOLDED")
}

func (s *APIServer) handlePlayerReady(w http.ResponseWriter, r *http.Request) error {
	s.game.SetReady()
	return JSON(w, http.StatusOK, "READY")
}
