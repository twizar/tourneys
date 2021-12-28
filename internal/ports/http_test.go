package ports_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/twizar/common/pkg/dto"
	"github.com/twizar/common/test/mock"
	"github.com/twizar/tourneys/internal/application/service"
	"github.com/twizar/tourneys/internal/ports"
	"github.com/twizar/tourneys/internal/ports/converter"
)

const (
	user1ID = "d9eaf6fb-5b61-48d0-bbe2-c5f4b912986a"
	user2ID = "a4990e03-9cb2-4b15-ae7a-73e1ea335ecc"
	user3ID = "a8b477c8-4040-4d11-a064-9a98c4b87131"
	user4ID = "a2cefbbc-b649-4abe-856c-faee4e7f62ce"

	liverpoolID = "d6548941-53f1-4d27-ad3d-0286cf512af1"
	milanID     = "5d912b4e-4932-496d-b706-c22b58f76a21"
	sevillaID   = "b0f6d915-da69-4681-bd7e-d933dd599ab2"
	bayernID    = "418ca28d-af10-4fbb-8b10-6afd74a001b7"
)

type generateTourneyCase struct {
	name                    string
	teamsServiceMockFactory func() *mock.MockTeams
	request                 ports.GenerateTourneyRequest
}

func case1(t *testing.T, ctrl *gomock.Controller, allTeams []dto.Team) generateTourneyCase {
	t.Helper()

	return generateTourneyCase{
		name: "Min rating 4, 32 teams",
		teamsServiceMockFactory: func() *mock.MockTeams {
			teamsService := mock.NewMockTeams(ctrl)
			data, err := os.ReadFile("../../test/data/search_teams_payload.js")
			require.NoError(t, err)
			var teamsSearch, teamsRequired []dto.Team
			err = json.Unmarshal(data, &teamsSearch)
			require.NoError(t, err)
			teamsService.EXPECT().SearchTeams(float64(3), []string{}, "rating", 0).Return(teamsSearch, nil)

			data, err = os.ReadFile("../../test/data/teams_by_ID_payload.js")
			require.NoError(t, err)
			err = json.Unmarshal(data, &teamsRequired)
			require.NoError(t, err)
			teamsService.EXPECT().TeamsByID([]string{liverpoolID, milanID, bayernID, sevillaID}).AnyTimes().Return(teamsRequired, nil)

			teamsService.EXPECT().TeamsByID(gomock.Any()).AnyTimes().AnyTimes().Return(allTeams, nil)

			return teamsService
		},
		request: ports.GenerateTourneyRequest{
			GroupsCount:   8,
			TeamsPerGroup: 4,
			Leagues:       []string{},
			Users: []ports.UserParams{
				{
					UserID:        user1ID,
					TeamsCount:    8,
					RequiredTeams: []string{liverpoolID},
				},
				{
					UserID:     user2ID,
					TeamsCount: 6,
				},
				{
					UserID:        user3ID,
					TeamsCount:    8,
					RequiredTeams: []string{milanID, bayernID},
				},
				{
					UserID:        user4ID,
					TeamsCount:    10,
					RequiredTeams: []string{sevillaID},
				},
			},
		},
	}
}

func case2(t *testing.T, ctrl *gomock.Controller, allTeams []dto.Team) generateTourneyCase {
	t.Helper()

	return generateTourneyCase{
		name: "Min rating 4, 16 teams",
		teamsServiceMockFactory: func() *mock.MockTeams {
			teamsService := mock.NewMockTeams(ctrl)
			data, err := os.ReadFile("../../test/data/search_teams_payload.js")
			require.NoError(t, err)
			var teamsSearch, teamsRequired []dto.Team
			err = json.Unmarshal(data, &teamsSearch)
			require.NoError(t, err)
			teamsService.EXPECT().SearchTeams(float64(3), []string{}, "rating", 0).Return(teamsSearch, nil)

			data, err = os.ReadFile("../../test/data/teams_by_ID_payload.js")
			require.NoError(t, err)
			err = json.Unmarshal(data, &teamsRequired)
			require.NoError(t, err)
			teamsService.EXPECT().TeamsByID([]string{liverpoolID}).AnyTimes().Return(teamsRequired, nil)

			teamsService.EXPECT().TeamsByID(gomock.Any()).AnyTimes().Return(allTeams, nil)

			return teamsService
		},
		request: ports.GenerateTourneyRequest{
			GroupsCount:   4,
			TeamsPerGroup: 4,
			Leagues:       []string{},
			Users: []ports.UserParams{
				{
					UserID:        user1ID,
					TeamsCount:    4,
					RequiredTeams: []string{liverpoolID},
				},
				{
					UserID:        user2ID,
					TeamsCount:    4,
					RequiredTeams: []string{milanID},
				},
				{
					UserID:        user3ID,
					TeamsCount:    4,
					RequiredTeams: []string{bayernID},
				},
				{
					UserID:        user4ID,
					TeamsCount:    4,
					RequiredTeams: []string{sevillaID},
				},
			},
		},
	}
}

func TestHTTPServer_GenerateTourney(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	t.Cleanup(func() {
		ctrl.Finish()
	})

	allTeamsData, err := os.ReadFile("../../test/data/teams.json")
	require.NoError(t, err)

	var allTeams []dto.Team

	err = json.Unmarshal(allTeamsData, &allTeams)
	require.NoError(t, err)

	tests := []struct {
		name                    string
		teamsServiceMockFactory func() *mock.MockTeams
		request                 ports.GenerateTourneyRequest
	}{
		case1(t, ctrl, allTeams),
		case2(t, ctrl, allTeams),
	}

	for _, testCase := range tests {
		testCase := testCase

		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			teamsService := testCase.teamsServiceMockFactory()
			tourneyGenerator := service.NewTourneyGenerator(teamsService)
			dtoConverter := converter.NewConverter(teamsService)
			server := ports.NewHTTPServer(tourneyGenerator, dtoConverter)
			router := ports.ConfigureRouter(server)

			tourneyParamsJSON, err := json.Marshal(testCase.request)
			require.NoError(t, err)

			request, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "/tourneys", bytes.NewBuffer(tourneyParamsJSON))
			require.NoError(t, err)

			writer := httptest.NewRecorder()
			router.ServeHTTP(writer, request)
			assert.Equal(t, http.StatusOK, writer.Code)

			body, err := io.ReadAll(writer.Body)
			require.NoError(t, err)

			var result []dto.Tourney
			err = json.Unmarshal(body, &result)
			require.NoError(t, err)
			assert.NotEmpty(t, result)
		})
	}
}
