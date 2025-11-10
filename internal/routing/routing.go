package routing

const (
	ArmyMovesPrefix = "army_moves"

	WarRecognitionsPrefix = "war"

	PauseKey     = "pause"
	ArmyMovesKey = "army_moves"

	GameLogSlug = "game_logs"

	DeadLetter = "peril_dlq"

	DeadLetterKey = "x-dead-letter-exchange"
)

const (
	ExchangePerilDirect = "peril_direct"
	ExchangePerilTopic  = "peril_topic"
	ExchangeDeadLetter  = "peril_dlx"
)
