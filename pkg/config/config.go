// Copyright (c) 2025 AccelByte Inc. All Rights Reserved.
// This is licensed software from AccelByte Inc, for limitations
// and restrictions contact your company contract manager.

package config

type Config struct {
	MatchTimeLimitSecond           int  `env:"MATCH_TIME_LIMIT_SECOND"            envDefault:"0"     envDocs:"configurable match time limit in second (0 means use default from code)"`
	FindAllyMaxLoop                int  `env:"FIND_ALLY_MAX_LOOP"                 envDefault:"0"     envDocs:"number of max loop in findMatchingAlly (0 means use default from code)"`
	FindPartyMaxLoop               int  `env:"FIND_PARTY_MAX_LOOP"                envDefault:"0"     envDocs:"number of max loop in FindPartyCombination (0 means use default from code)"`
	PrioritizeLargerParties        bool `env:"PRIORITIZE_LARGER_PARTIES"          envDefault:"false" envDocs:"prioritize larger parties during find matches"`
	FlagAnyMatchOptionAllCommon    bool `env:"FLAG_ANY_MATCH_OPTION_ALL_COMMON"   envDefault:"true"  envDocs:"Any match option match common value for all tickets, not only by pivot ticket"`
	CrossPlatformNoCurrentPlatform bool `env:"CROSS_PLATFORM_NO_CURRENT_PLATFORM" envDefault:"false" envDocs:"If true current_platform attribute won't be added when creating match ticket. Can be overridden on namespace config."`
}
