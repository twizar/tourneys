package ports

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/twizar/tourneys/internal/application/service"
	"github.com/twizar/tourneys/internal/domain/entity"
	"github.com/twizar/tourneys/internal/ports/converter"
)

type UserParams struct {
	UserID        string   `json:"user_id"`
	TeamsCount    int      `json:"teams_count"`
	RequiredTeams []string `json:"required_teams"`
}

type GenerateTourneyRequest struct {
	GroupsCount   int          `json:"groups_count"`
	TeamsPerGroup int          `json:"teams_per_group"`
	Leagues       []string     `json:"leagues"`
	Users         []UserParams `json:"users"`
}

type HTTPServer struct {
	tourneyGenerator *service.TourneyGenerator
	dtoConverter     *converter.Converter
}

func NewHTTPServer(tourneyGenerator *service.TourneyGenerator, dtoConverter *converter.Converter) *HTTPServer {
	return &HTTPServer{tourneyGenerator: tourneyGenerator, dtoConverter: dtoConverter}
}

func (s HTTPServer) GenerateTourney(writer http.ResponseWriter, request *http.Request) {
	tourneyRequest := new(GenerateTourneyRequest)
	if err := json.NewDecoder(request.Body).Decode(&tourneyRequest); err != nil {
		http.Error(writer, "bad tourney request payload", http.StatusBadRequest)
		log.Printf("tourney request payload error: %v\n", err)

		return
	}

	usersSettings := make([]*service.UserSettingsDTO, len(tourneyRequest.Users))
	for i, userParams := range tourneyRequest.Users {
		usersSettings[i] = service.NewUserSettingsDTO(userParams.UserID, userParams.TeamsCount, userParams.RequiredTeams)
	}

	tourney, err := s.tourneyGenerator.Generate(
		tourneyRequest.GroupsCount,
		tourneyRequest.TeamsPerGroup,
		tourneyRequest.Leagues,
		usersSettings,
	)
	if err != nil {
		http.Error(writer, "tourney generation error", http.StatusBadRequest)
		log.Printf("tourney generation error: %v\n", err)

		return
	}

	tourneyDTOs, err := s.dtoConverter.TourneyEntitiesToDTOs([]*entity.Tourney{tourney})
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		log.Printf("converting tourney entity to DTO error: %v\n", err)

		return
	}

	err = json.NewEncoder(writer).Encode(tourneyDTOs)
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
		log.Printf("encoding response error: %v\n", err)

		return
	}

	writer.WriteHeader(http.StatusOK)
}

func ConfigureRouter(server *HTTPServer) *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/tourneys", server.GenerateTourney).Methods(http.MethodPost)

	return router
}
