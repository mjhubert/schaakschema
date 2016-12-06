package main

import "fmt"

//Speelschema
type SpeelSchema struct {
}

//Klasse van Schaken
type Klasse int

const (
	//Meester klasse
	Meester Klasse = iota
	//Eerste klasse
	Eerste
	//Tweede klasse
	Tweede
	//Derde klasse
	Derde
)

//Gradatie wijziging
type Gradatie int

const (
	//Ongewijzigd gradatie
	Ongewijzigd Gradatie = iota
	//Promotie gradatie
	Promotie
	//Degradatie gradatie
	Degradatie
	//Kampioen gradatie
	Kampioen
)

//Team van een vereniging
type Team struct {
	id, naam   string
	klasse     Klasse
	pd         Gradatie
	vereniging Vereniging
}

func (x Team) String() string {
	return fmt.Sprintf("{ id: %s, naam: %s, klasse: %v, pd: %v, vereniging: %s}", x.id, x.naam, x.klasse, x.pd, x.vereniging.id)
}

//Vereniging van de Schaakbond
type Vereniging struct {
	id, naam, plaats string
	teams            map[string]Team
}

func (x Vereniging) String() string {
	return fmt.Sprintf("{ id: %s, naam: %s, plaats: %s, teams: %d}", x.id, x.naam, x.plaats, len(x.teams))
}

//Schaakbond van Nederland
type Schaakbond struct {
	verenigingen map[string]Vereniging
	teams        map[string]Team
}
