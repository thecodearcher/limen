package jwt

type JWTAlgorithm string

// jwt algorithms
const (
	JWTAlgorithmHS256 JWTAlgorithm = "HS256"
	JWTAlgorithmHS384 JWTAlgorithm = "HS384"
	JWTAlgorithmHS512 JWTAlgorithm = "HS512"
	JWTAlgorithmRS256 JWTAlgorithm = "RS256"
	JWTAlgorithmRS384 JWTAlgorithm = "RS384"
	JWTAlgorithmRS512 JWTAlgorithm = "RS512"
	JWTAlgorithmES256 JWTAlgorithm = "ES256"
	JWTAlgorithmES384 JWTAlgorithm = "ES384"
	JWTAlgorithmES512 JWTAlgorithm = "ES512"
)
