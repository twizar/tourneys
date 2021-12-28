package service

import (
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"time"

	goFunk "github.com/thoas/go-funk"
	"github.com/twizar/common/pkg/client"
	"github.com/twizar/common/pkg/dto"
	"github.com/twizar/tourneys/internal/domain/entity"
)

type (
	UserID          string
	RequiredTeamIDs []string
)

const (
	minRatingDefault = 3
	minLimitDefault  = 0
)

var errTypeAssertion = errors.New("type assertion error")

type UserSettingsDTO struct {
	userID          string
	teamsCount      int
	requiredTeamIDs []string
}

func NewUserSettingsDTO(userID string, teamsCount int, requiredTeamIDs []string) *UserSettingsDTO {
	return &UserSettingsDTO{userID: userID, teamsCount: teamsCount, requiredTeamIDs: requiredTeamIDs}
}

type TourneyGenerator struct {
	teams client.Teams
}

func NewTourneyGenerator(teams client.Teams) *TourneyGenerator {
	return &TourneyGenerator{teams: teams}
}

func (tg TourneyGenerator) Generate(
	groupsCount,
	teamsPerGroup int,
	leagues []string,
	usersSettings []*UserSettingsDTO,
) (*entity.Tourney, error) {
	teams, err := tg.teams.SearchTeams(minRatingDefault, leagues, "rating", minLimitDefault)
	if err != nil {
		return nil, fmt.Errorf("searching teams error: %w", err)
	}

	requiredTeamIDs := requiredTeamIDsFromUserSettings(usersSettings)

	requiredTeams, err := tg.teams.TeamsByID(requiredTeamIDs)
	if err != nil {
		return nil, fmt.Errorf("getting teams by ID error: %w", err)
	}

	var ok bool

	requiredTeamsGroupedByUserID, err := groupTeams(usersSettings, requiredTeams)
	if err != nil {
		return nil, fmt.Errorf("group teams error: %w", err)
	}

	if teams, ok = goFunk.Filter(teams, func(team dto.Team) bool {
		return !goFunk.ContainsString(requiredTeamIDs, team.ID)
	}).([]dto.Team); !ok {
		return nil, fmt.Errorf("DTO teams slice assertion error: %w", errTypeAssertion)
	}

	groupedByRatingTeams := groupTeamsByRating(teams)

	groups := generateShuffledGroups(groupsCount, teamsPerGroup)

	var team dto.Team

	emitTeamForUser := func(userID string) dto.Team {
		if len(requiredTeamsGroupedByUserID[userID]) > 0 {
			shuffleTeams(requiredTeamsGroupedByUserID[userID])
			team, requiredTeamsGroupedByUserID[userID] = popTeam(requiredTeamsGroupedByUserID[userID])

			return team
		}

		for index := range groupedByRatingTeams {
			if len(groupedByRatingTeams[index]) == 0 {
				continue
			}

			shuffleTeams(groupedByRatingTeams[index])
			team, groupedByRatingTeams[index] = popTeam(groupedByRatingTeams[index])

			break
		}

		return team
	}

	fillGroups(groups, groupsCount, teamsPerGroup, usersSettings, emitTeamForUser)

	groups = normalizeGroups(groups)

	return entity.NewTourney("", groupsCount, teamsPerGroup, groups), nil
}

func fillGroups(
	groups []*entity.Group,
	groupsCount,
	teamsPerGroup int,
	usersSettings []*UserSettingsDTO,
	emitTeamForUser func(userID string) dto.Team,
) {
	currentPointer := 0
	emitUserID := func() string {
		if currentPointer == len(usersSettings) {
			currentPointer = 0
		}

		userID := usersSettings[currentPointer].userID
		currentPointer++

		return userID
	}

	isGroupFullFilled := func(group *entity.Group) bool {
		for i := 0; i < teamsPerGroup; i++ {
			if group.TeamSlots()[i] == nil {
				return false
			}
		}

		return true
	}

	teamsCount := groupsCount * teamsPerGroup

	slotsBucket := make([]*entity.GroupSlot, teamsCount)

	for i := 0; i < teamsCount; i++ {
		userID := emitUserID()
		teamID := emitTeamForUser(userID).ID
		slotsBucket[i] = entity.NewGroupSlot(userID, teamID)
	}

	currentEmittedUserIndex := 0
	emitSlot := func() *entity.GroupSlot {
		user := usersSettings[currentEmittedUserIndex]
		currentEmittedUserIndex++

		if currentEmittedUserIndex == len(usersSettings) {
			currentEmittedUserIndex = 0
		}

		shuffleSlots(slotsBucket)

		for i := range slotsBucket {
			if slotsBucket[i].UserID() == user.userID {
				slot := slotsBucket[i]
				slotsBucket = removeSlotByIndex(slotsBucket, i)

				return slot
			}
		}

		return nil
	}

	for i := range groups {
		currentSlotsIndex := 0

		for !isGroupFullFilled(groups[i]) {
			slot := emitSlot()
			groups[i].AssignSlot(currentSlotsIndex, slot)
			currentSlotsIndex++
		}
	}
}

func groupTeams(usersSettings []*UserSettingsDTO, requiredTeams []dto.Team) (map[string][]dto.Team, error) {
	requiredTeamsGroupedByUserID := make(map[string][]dto.Team, len(usersSettings))

	var ok bool

	for _, settings := range usersSettings {
		settings := settings

		filteredTeams := goFunk.Filter(requiredTeams, func(team dto.Team) bool {
			return goFunk.ContainsString(settings.requiredTeamIDs, team.ID)
		})

		if requiredTeamsGroupedByUserID[settings.userID], ok = filteredTeams.([]dto.Team); !ok {
			return nil, fmt.Errorf("DTO teams slice assertion error: %w", errTypeAssertion)
		}
	}

	return requiredTeamsGroupedByUserID, nil
}

func requiredTeamIDsFromUserSettings(usersSettings []*UserSettingsDTO) []string {
	var requiredTeamIDs []string
	for _, settings := range usersSettings {
		requiredTeamIDs = append(requiredTeamIDs, settings.requiredTeamIDs...)
	}

	return requiredTeamIDs
}

func generateShuffledGroups(count, teamsPerGroup int) []*entity.Group {
	groups := make([]*entity.Group, count)
	startASCII := 97

	for i := 0; i < count; i++ {
		name := string(rune(startASCII + i))
		groups[i] = entity.NewGroup(name, make([]*entity.GroupSlot, teamsPerGroup))
	}

	rand.Seed(time.Now().UnixMicro())
	rand.Shuffle(len(groups), func(i, j int) {
		groups[i], groups[j] = groups[j], groups[i]
	})

	return groups
}

func shuffleSlots(slots []*entity.GroupSlot) {
	rand.Seed(time.Now().UnixMicro())
	rand.Shuffle(len(slots), func(i, j int) {
		slots[i], slots[j] = slots[j], slots[i]
	})
}

func shuffleTeams(teams []dto.Team) {
	rand.Seed(time.Now().UnixMicro())
	rand.Shuffle(len(teams), func(i, j int) {
		teams[i], teams[j] = teams[j], teams[i]
	})
}

func popTeam(teams []dto.Team) (team dto.Team, poppedTeams []dto.Team) {
	return teams[len(teams)-1], teams[:len(teams)-1]
}

func removeSlotByIndex(slots []*entity.GroupSlot, i int) []*entity.GroupSlot {
	slots[i] = slots[len(slots)-1]

	return slots[:len(slots)-1]
}

func groupTeamsByRating(teams []dto.Team) [][]dto.Team {
	tmp := make(map[float64][]dto.Team)
	for _, team := range teams {
		tmp[team.Rating] = append(tmp[team.Rating], team)
	}

	keys := make([]float64, 0, len(tmp))
	for k := range tmp {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i] > keys[j]
	})

	groupedTeams := make([][]dto.Team, len(tmp))
	for i, k := range keys {
		groupedTeams[i] = tmp[k]
	}

	return groupedTeams
}

func normalizeGroups(groups []*entity.Group) []*entity.Group {
	for i := range groups {
		shuffleSlots(groups[i].TeamSlots())
	}

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Name() < groups[j].Name()
	})

	return groups
}
