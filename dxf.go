package dxf

//go:generate go run pregenerate/copyCodePairHelper.go
//go:generate go run generator/generate.go generator/generatorHelpers.go generator/codePairHelper.go generator/entityGenerator.go generator/enumGenerator.go generator/headerGenerator.go generator/tableGenerator.go
