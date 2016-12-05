package main

import (
	"fmt"
	"log"

	"github.com/tealeg/xlsx"
)

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

//LoadSchaakbondExcel Laad teams en verenigingen uit het excel-bestand
func LoadSchaakbondExcel(fileName string) Schaakbond {
	xlFile, err := xlsx.OpenFile(fileName)

	if err != nil {
		log.Fatal(err)
	}

	var sb Schaakbond
	sb.verenigingen = make(map[string]Vereniging)
	sb.teams = make(map[string]Team)

	for _, sheet := range xlFile.Sheets {
		if sheet.Name == "Indeling" {
			for _, row := range sheet.Rows {
				//0 - Team Id
				//1 - Team Klasse { M, 1, 2, 3 }
				//4 - Team Naam
				//5 - Vereniging Id
				//6 - Vereniging plaats
				//7 - Team D/P/_

				if row.Cells[0].Value != "" &&
					row.Cells[0].Value != "Teamid" {

					v, ok := sb.verenigingen[row.Cells[5].Value]

					if !ok {
						var nw Vereniging
						nw.id = row.Cells[5].Value
						nw.plaats = row.Cells[6].Value
						nw.teams = make(map[string]Team)
						sb.verenigingen[nw.id] = nw
						v = nw
					}

					var t Team
					t.id = row.Cells[0].Value
					t.naam = row.Cells[4].Value

					switch row.Cells[1].Value {
					case "M":
						t.klasse = Meester
					case "1":
						t.klasse = Eerste
					case "2":
						t.klasse = Tweede
					case "3":
						t.klasse = Derde
					default:
						log.Panic("Unknown Klasse value of team ", row.Cells[0].Value, row.Cells[1].Value)
					}

					switch row.Cells[7].Value {
					case "K":
						t.pd = Kampioen
					case "P":
						t.pd = Promotie
					case "D":
						t.pd = Degradatie
					case "":
						t.pd = Ongewijzigd
					default:
						log.Panic("Unknown P/D value of team ", row.Cells[0].Value, row.Cells[7].Value)
					}

					t.vereniging = v
					v.teams[t.id] = t
					sb.teams[t.id] = t

				}
			}
		}
	}

	return sb
}
