// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package matchmaker

import (
	"sync"
	"time"
)

// WorkerInfo stores the worker info
type WorkerInfo struct {
	Timestamp                   time.Time `json:"timestamp"`
	Namespace                   string    `json:"namespace"`
	Matchpool                   string    `json:"matchpool"`
	PodName                     string    `json:"podname"`
	TickID                      int64     `json:"tickID"`
	MatchCreated                int       `json:"matchCreated"`
	TotalTicketForBackfill      int       `json:"totalTicketForBackfill"`
	TotalTicketBackfillSuccess  int       `json:"totalTicketBackfillSuccess"`
	TotalTicketForMatch         int       `json:"totalTicketForMatch"`
	TotalTicketMatchSuccess     int       `json:"totalTicketMatchSuccess"`
	TotalMatchToBackfill        int       `json:"totalMatchToBackfill"`
	TotalMatchBackfilledSuccess int       `json:"totalMatchBackfilledSuccess"`
	TotalTicketInQueue          int       `json:"totalTicketInQueue"`

	Mutex sync.Mutex `json:"-"`

	// GlobalPoolCount is used to count delay = max_delay_ms / GlobalPoolCount,
	// it is not exported because it's only count global pool before the worker starts getting tickets,
	// it is not count the final global pool when match is created
	GlobalPoolCount int `json:"-"`

	// for local usage only
	ShouldSend bool `json:"-"`
}
