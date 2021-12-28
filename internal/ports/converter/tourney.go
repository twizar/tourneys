package converter

import (
	"errors"
	"fmt"

	"github.com/twizar/common/pkg/client"
	"github.com/twizar/common/pkg/dto"
	"github.com/twizar/tourneys/internal/domain/entity"
)

var errTeamNotFoundInStorage = errors.New("team hasn't been found in storage")

type Converter struct {
	teams        client.Teams
	teamsStorage map[string]dto.Team
}

func NewConverter(teams client.Teams) *Converter {
	return &Converter{teams: teams, teamsStorage: make(map[string]dto.Team)}
}

func (c Converter) TourneyEntitiesToDTOs(entityTourneys []*entity.Tourney) ([]dto.Tourney, error) {
	dtoTourneys := make([]dto.Tourney, len(entityTourneys))

	ids := fetchTeamsIDFromTourneysCollection(entityTourneys)

	err := c.fillTeamsStorage(ids)
	if err != nil {
		return nil, fmt.Errorf("refreshing teams cache error: %w", err)
	}

	for index, tourney := range entityTourneys {
		groupDTOs, err := c.groupEntitiesToDTOs(tourney.Groups())
		if err != nil {
			return nil, fmt.Errorf("converting groups error: %w", err)
		}

		dtoTourneys[index] = dto.Tourney{
			ID:            tourney.ID(),
			GroupsCount:   tourney.GroupsCount(),
			TeamsPerGroup: tourney.TeamsPerGroup(),
			Groups:        groupDTOs,
		}
	}

	return dtoTourneys, nil
}

func (c Converter) groupEntitiesToDTOs(entityGroups []*entity.Group) ([]dto.Group, error) {
	dtoTeams := make([]dto.Group, len(entityGroups))

	for index, group := range entityGroups {
		groupDTOs, err := c.groupSlotsEntitiesToDTOs(group.TeamSlots())
		if err != nil {
			return nil, fmt.Errorf("converting slots error: %w", err)
		}

		dtoTeams[index] = dto.Group{
			Name:      group.Name(),
			TeamSlots: groupDTOs,
		}
	}

	return dtoTeams, nil
}

func (c Converter) groupSlotsEntitiesToDTOs(entitySlots []*entity.GroupSlot) ([]dto.GroupSlot, error) {
	dtoTeams := make([]dto.GroupSlot, len(entitySlots))

	for index, slot := range entitySlots {
		team, ok := c.teamsStorage[slot.TeamID()]
		if !ok {
			return nil, fmt.Errorf("team` %s` not found %w", slot.TeamID(), errTeamNotFoundInStorage)
		}

		dtoTeams[index] = dto.GroupSlot{
			UserID: slot.UserID(),
			Team:   team,
		}
	}

	return dtoTeams, nil
}

func fetchTeamsIDFromTourneysCollection(tourneys []*entity.Tourney) (ids []string) {
	for _, tourney := range tourneys {
		for _, group := range tourney.Groups() {
			for _, slot := range group.TeamSlots() {
				ids = append(ids, slot.TeamID())
			}
		}
	}

	return
}

func (c *Converter) fillTeamsStorage(ids []string) error {
	teams, err := c.teams.TeamsByID(ids)
	if err != nil {
		return fmt.Errorf("getting teams by ID error: %w", err)
	}

	c.teamsStorage = make(map[string]dto.Team, len(teams))

	for _, team := range teams {
		c.teamsStorage[team.ID] = team
	}

	return nil
}
