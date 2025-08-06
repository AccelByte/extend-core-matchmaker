# Custom MatchMaker - Basic

## Summary
The matchmaker.go file is an example of a Basic Custom MatchMaker that will match 2 tickets
without any checks. Below are descriptions of what each of these MatchMaker's functions are do.

### ValidateTicket()
Does not perform a validation check and returns `true`, making the ticket valid.

### EnrichTicket()
Checks if the match ticket's `TicketAttributes` map is empty -- if so, it will add in a
key-value number.

### GetStatCodes()
Returns an empty string slice.

### RulesFromJSON()
Unmarshals the json rules string to the appropriate ruleSet `(GameRules)` and returns them
as an interface.

### MakeMatches()
Creates a `results` go channel and invokes `GetTickets()` from the `TicketProvider` interface. Next,
select each ticket and (if there are tickets in the channel) call `buildMatch`, which will dumbly
match any 2 tickets and send them to the created `results` channel.