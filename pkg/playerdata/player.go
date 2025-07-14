// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package playerdata

// PlayerData is an id for a playerdata along with a collection of arbitrary data used for matching this playerdata with others.
type PlayerData struct {
	PlayerID   ID
	PartyID    string
	Attributes map[string]interface{}
}

// ID is a unique identifier for a playerdata.
type ID string

// IDToString returns a string representation of the playerdata id.
func IDToString(u ID) string {
	return string(u)
}

// IDFromString converts a string to a playerdata.ID.
func IDFromString(u string) ID {
	return ID(u)
}

func ToID(p PlayerData) ID {
	return p.PlayerID
}

func ToIDString(p PlayerData) string {
	return string(p.PlayerID)
}
