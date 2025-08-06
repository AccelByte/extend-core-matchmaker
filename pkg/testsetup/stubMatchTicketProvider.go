// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package testsetup

import (
	"time"

	"github.com/AccelByte/extend-core-matchmaker/pkg/matchmaker"
)

type StubMatchTicketProvider struct {
	Tickets         []matchmaker.Ticket
	BackfillTickets []matchmaker.BackfillTicket
	PerTicketDelay  time.Duration
}

func (s StubMatchTicketProvider) GetBackfillTickets() chan matchmaker.BackfillTicket {
	ticketChannel := make(chan matchmaker.BackfillTicket)
	go func() {
		for _, ticket := range s.BackfillTickets {
			time.Sleep(s.PerTicketDelay)
			ticketChannel <- ticket
		}
		close(ticketChannel)
	}()
	return ticketChannel
}

func (s StubMatchTicketProvider) GetTickets() chan matchmaker.Ticket {
	ticketChannel := make(chan matchmaker.Ticket)
	go func() {
		for _, ticket := range s.Tickets {
			time.Sleep(s.PerTicketDelay)
			ticketChannel <- ticket
		}
		close(ticketChannel)
	}()
	return ticketChannel
}

func (s StubMatchTicketProvider) Count() int64 {
	return 0
}

func (s StubMatchTicketProvider) BackfillTicketCount() int64 {
	return 0
}

func (s StubMatchTicketProvider) UnclaimTicket(workerID string) error {
	return nil
}

func (s StubMatchTicketProvider) UnclaimBackfillTicket(workerID string) error {
	return nil
}

func (s StubMatchTicketProvider) GetTicketByWorkerID(workerID string) ([]matchmaker.Ticket, error) {
	return nil, nil
}

func (s StubMatchTicketProvider) GetBackfillTicketByWorkerID(workerID string) ([]matchmaker.BackfillTicket, error) {
	return nil, nil
}
