package chains

func NewAuthorizationTestChain() (ChainResult) {
	// check that there is at least one registration or return empty CR so attacker can continue
	var cR ChainResult

	cR.Name = "new authorization"

	return cR
}