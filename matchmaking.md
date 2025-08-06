# Matchmaking System Documentation

## Overview

The AccelByte Extend Core Matchmaker is a sophisticated matchmaking system designed to efficiently pair players into optimal game sessions. The system supports both new match creation and backfill operations for existing sessions, with configurable rules and dynamic adjustments based on queue time.

## Architecture

### Core Components

#### 1. **MatchLogic Interface** (`pkg/matchmaker/interfaces.go`)
The primary interface that defines the contract for matchmaking implementations:

- **MakeMatches**: Creates new matches from available tickets
- **BackfillMatches**: Adds players to existing sessions
- **RulesFromJSON**: Parses configuration rules
- **ValidateTicket**: Validates incoming matchmaking requests
- **EnrichTicket**: Adds computed data to tickets

#### 2. **Data Structures** (`pkg/matchmaker/dataobjects.go`)

**Ticket**: Represents a matchmaking request
```go
type Ticket struct {
    Namespace        string                 // Tenant namespace
    PartySessionID   string                 // Party identifier
    TicketID         string                 // Unique ticket ID
    MatchPool        string                 // Target match pool
    CreatedAt        time.Time              // Creation timestamp
    Players          []playerdata.PlayerData // Player list
    TicketAttributes map[string]interface{} // Custom attributes
    Latencies        map[string]int64       // Region latencies
    ExcludedSessions []string               // Excluded sessions
}
```

**Match**: Represents a matchmaking result
```go
type Match struct {
    Tickets                      []Ticket                    // Matched tickets
    Teams                        []Team                      // Team assignments
    RegionPreference             []string                    // Preferred regions
    MatchAttributes              map[string]interface{}      // Match attributes
    Backfill                     bool                        // Backfill flag
    ServerName                   string                      // Target server
    ClientVersion                string                      // Game version
    ServerPoolSelectionParameter ServerPoolSelectionParameter // Server selection
    PivotID                      string                      // Pivot ticket ID
    Timestamp                    time.Time                   // Match timestamp
}
```

#### 3. **Default Implementation** (`pkg/matchmaker/defaultmatchmaker/`)
The default matchmaking implementation providing:

- **Pivot-based matching** algorithm
- **Rule flexing** for aging tickets
- **Region-based** latency optimization
- **Party combination** logic
- **Backfill** operations

## Matchmaking Flow

### 1. **Ticket Processing**

When a matchmaking request arrives:

1. **Validation**: `ValidateTicket()` checks if the request meets basic requirements
2. **Enrichment**: `EnrichTicket()` adds computed attributes and validates data
3. **Queueing**: Ticket is added to the appropriate match pool

### 2. **New Match Creation** (`MakeMatches`)

The core matchmaking process follows these steps:

#### Step 1: Ticket Preparation
```go
// Convert tickets to internal format
requests := toMatchRequest(tickets, ruleSet)
```

#### Step 2: Pivot Selection
- Selects the oldest ticket as the pivot
- Applies rule flexing based on pivot age
- Determines active matching criteria

#### Step 3: Candidate Search
```go
// Find compatible tickets using multiple criteria
candidates := SearchMatchTickets(originalRuleSet, activeRuleSet, channel, regionIndex, pivot, tickets, filteredRegion)
```

**Search Criteria:**
- **Distance-based matching**: MMR, skill level, etc.
- **Match options**: Cross-play, game modes, etc.
- **Party attributes**: Server preferences, client versions
- **Blocked players**: Player exclusion lists
- **Region latency**: Network performance optimization

#### Step 4: Alliance Formation
```go
// Find optimal team combinations
allies := findMatchingAlly(scope, config, sourceTickets, pivotTicket, allianceRule, matchingRules, blockedPlayerOption)
```

**Alliance Logic:**
- Creates teams based on `AllianceRule` configuration
- Ensures minimum/maximum player counts per team
- Balances team sizes optimally
- Handles role-based assignments if configured

#### Step 5: Match Creation
```go
// Convert results to match objects
match := fromMatchResult(result, sourceTickets, ruleSet)
```

### 3. **Backfill Operations** (`BackfillMatches`)

For existing sessions that need more players:

#### Step 1: Session Analysis
- Identifies sessions requiring backfill
- Determines missing player counts
- Applies session-specific rule flexing

#### Step 2: Ticket Matching
```go
// Find tickets compatible with existing session
candidates := SearchMatchTicketsBySession(scope, originalRuleSet, activeRuleSet, channel, session, tickets)
```

**Session Matching Criteria:**
- **Region compatibility**: Must match session region
- **Server compatibility**: Must use same server
- **Version compatibility**: Must use same client version
- **Team balance**: Must fit within team constraints
- **Attribute compatibility**: Must match session attributes

#### Step 3: Session Update
- Adds matched tickets to existing teams
- Updates session attributes and player counts
- Removes full sessions from backfill pool

## Key Algorithms

### 1. **Pivot-Based Matching**

The system uses a pivot-based approach where the oldest ticket becomes the reference point:

```go
// Select pivot (oldest ticket)
pivotTicket := requests[0]

// Apply rule flexing based on pivot age
activeRuleSet, _ := applyRuleFlexing(ruleSet, pivotTime)

// Find compatible tickets
candidates := SearchMatchTickets(originalRuleSet, activeRuleSet, channel, regionIndex, pivotTicket, tickets, filteredRegion)
```

**Benefits:**
- Ensures fair queue processing (FIFO)
- Enables rule flexing for aging tickets
- Provides consistent matching behavior

### 2. **Rule Flexing**

As tickets age in the queue, matching criteria become more lenient:

```go
func applyRuleFlexing(sourceRuleSet models.RuleSet, pivotTime time.Time) (models.RuleSet, bool) {
    // Find active flex rules based on ticket age
    for _, flexRule := range ruleset.FlexingRule {
        if isActiveFlexRule(pivotTime, flexDuration) {
            // Apply more lenient criteria
            ruleset.MatchingRule[i].Reference = flexRule.Reference
            ruleset.MatchingRule[i].Criteria = flexRule.Criteria
        }
    }
}
```

**Flexing Types:**
- **Distance flexing**: Widens acceptable skill/MMR ranges
- **Alliance flexing**: Adjusts team size requirements
- **Latency flexing**: Expands acceptable region ranges

### 3. **Region Expansion**

The system optimizes for latency by expanding region search:

```go
func filterRegionByStep(ticket models.MatchmakingRequest, channel models.Channel) []models.Region {
    // Start with best latency regions
    // Expand to additional regions based on configuration
    // Apply latency thresholds and preferences
}
```

**Expansion Strategy:**
1. Start with lowest latency regions
2. Expand to additional regions if needed
3. Apply maximum latency thresholds
4. Consider region preferences and restrictions

### 4. **Party Combination**

For team-based games, the system finds optimal party combinations:

```go
func FindPartyCombination(config *config.Config, tickets []models.MatchmakingRequest, pivot models.MatchmakingRequest, minPlayer, maxPlayer int, current []models.MatchmakingRequest, blockedPlayerOption models.BlockedPlayerOption) []models.MatchmakingRequest {
    // Use PartyFinder to find optimal combinations
    pf := GetPartyFinder(minPlayer, maxPlayer, current)
    
    // Try different ticket orderings
    for r.HasNext() {
        newIndexes := r.Get()
        tickets := reorderTickets(sourceTickets, newIndexes)
        
        // Attempt to assign tickets to current combination
        for _, ticket := range tickets {
            if pf.AssignMembers(ticket) {
                pf.AppendResult(ticket)
                if pf.IsFulfilled() {
                    return pf.GetBestResult()
                }
            }
        }
    }
}
```

## Configuration System

### RuleSet Structure

Matchmaking behavior is controlled by JSON configuration:

```json
{
  "allianceRule": {
    "minNumber": 2,
    "maxNumber": 4,
    "playerMinNumber": 1,
    "playerMaxNumber": 2
  },
  "matchingRule": [
    {
      "attribute": "mmr",
      "criteria": "distance",
      "reference": 100,
      "weight": 1.0,
      "normalizationMax": 2000
    }
  ],
  "flexingRule": [
    {
      "attribute": "mmr",
      "criteria": "distance",
      "reference": 200,
      "duration": 30
    }
  ],
  "matchOptions": {
    "options": [
      {
        "name": "cross_platform",
        "type": "any"
      }
    ]
  }
}
```

### Key Configuration Elements

#### **AllianceRule**
- `minNumber`/`maxNumber`: Team count range
- `playerMinNumber`/`playerMaxNumber`: Players per team range

#### **MatchingRule**
- `attribute`: Player attribute to match on
- `criteria`: Matching algorithm (distance, exact, etc.)
- `reference`: Matching tolerance or exact value
- `weight`: Scoring weight for this rule
- `normalizationMax`: Maximum value for score normalization

#### **FlexingRule**
- `duration`: Seconds before flexing activates
- `reference`: New tolerance value after flexing

#### **MatchOptions**
- `name`: Option attribute name
- `type`: Matching type (any, all, unique, disable)

## Performance Optimizations

### 1. **Efficient Search**

The system uses multiple optimization techniques:

- **Early termination**: Stop searching when criteria don't match
- **Indexed lookups**: Use maps for O(1) blocked player checks
- **Sorted processing**: Process tickets in priority order
- **Batch operations**: Process multiple tickets simultaneously

### 2. **Memory Management**

- **Object reuse**: Reuse data structures where possible
- **Lazy evaluation**: Only compute values when needed
- **Efficient data structures**: Use appropriate containers for different operations

### 3. **Concurrency**

- **Parallel processing**: Use goroutines for independent operations
- **Channel-based communication**: Efficient result passing
- **WaitGroup synchronization**: Coordinate parallel operations

## Monitoring and Observability

### Logging

Comprehensive logging is provided through the `envelope.Scope`:

```go
scope.Log.WithField("ticket_id", ticket.TicketID).
    WithField("match_pool", ticket.MatchPool).
    Info("Processing matchmaking ticket")
```

### Attributes

The system sets attributes for external monitoring:

```go
scope.SetAttribute("tickets_processed", len(tickets))
scope.SetAttribute("matches_created", len(matches))
scope.SetAttribute("backfill_operations", backfillCount)
```

## Error Handling

### Validation Errors

The system validates tickets before processing:

- **Invalid attributes**: Missing required fields
- **Invalid ranges**: Values outside acceptable bounds
- **Invalid combinations**: Conflicting requirements

### Recovery Mechanisms

- **Graceful degradation**: Continue processing with valid tickets
- **Error reporting**: Detailed error messages for debugging
- **Fallback behavior**: Use default values when possible

## Troubleshooting

### Common Issues

1. **High Queue Times**
   - Check if rules are too restrictive
   - Verify region configurations
   - Monitor server capacity

2. **Poor Match Quality**
   - Review matching criteria
   - Adjust rule weights
   - Check attribute distributions

3. **Backfill Failures**
   - Verify session compatibility
   - Check region restrictions
   - Review team size constraints

### Debugging Tools

- **Detailed logging**: Enable debug logging for specific components
- **Configuration validation**: Verify rule configurations
- **Test scenarios**: Use test data to reproduce issues
