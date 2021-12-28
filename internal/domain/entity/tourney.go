package entity

type GroupSlot struct {
	userID,
	teamID string
}

func NewGroupSlot(userID, teamID string) *GroupSlot {
	return &GroupSlot{userID: userID, teamID: teamID}
}

func (u GroupSlot) UserID() string {
	return u.userID
}

func (u GroupSlot) TeamID() string {
	return u.teamID
}

type Group struct {
	name      string
	teamSlots []*GroupSlot
}

func NewGroup(name string, teamSlots []*GroupSlot) *Group {
	return &Group{name: name, teamSlots: teamSlots}
}

func (g Group) Name() string {
	return g.name
}

func (g *Group) AssignSlot(slotIndex int, slot *GroupSlot) {
	g.teamSlots[slotIndex] = slot
}

func (g Group) TeamSlots() []*GroupSlot {
	return g.teamSlots
}

type Tourney struct {
	id            string
	groupsCount   int
	teamsPerGroup int
	groups        []*Group
}

func (t Tourney) ID() string {
	return t.id
}

func (t Tourney) GroupsCount() int {
	return t.groupsCount
}

func (t Tourney) TeamsPerGroup() int {
	return t.teamsPerGroup
}

func (t Tourney) Groups() []*Group {
	return t.groups
}

func NewTourney(id string, groupsCount, teamsPerGroup int, groups []*Group) *Tourney {
	return &Tourney{id: id, groupsCount: groupsCount, teamsPerGroup: teamsPerGroup, groups: groups}
}
