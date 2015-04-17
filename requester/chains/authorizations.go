package chains

func NewAuthorizationTestChain(cC ChainContext) (ChainResult, ChainContext) {
	cR := ChainResult{Name: "new authorization"}

	return cR, cC
}