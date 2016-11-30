package main

import (
	"log"

	"github.com/tealeg/xlsx"
)

//Team van een vereniging
type Team struct {
	id, naam, klasse, pd string
	vereniging           Vereniging
}

//Vereniging van de Schaakbond
type Vereniging struct {
	id, naam, plaats string
	teams            map[string]Team
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
				//6 - Vereniging Naam
				//7 - Vereniging plaats
				//8 - Team D/P/_

				if row.Cells[0].Value != "" &&
					row.Cells[0].Value != "Teamid" {

					v, ok := sb.verenigingen[row.Cells[5].Value]

					if !ok {
						var nw Vereniging
						nw.id = row.Cells[5].Value
						nw.naam = row.Cells[6].Value
						nw.plaats = row.Cells[7].Value
						nw.teams = make(map[string]Team)
						sb.verenigingen[v.id] = nw
						v = nw

						log.Printf("Vereniging: %v", v)
					}

					var t Team
					t.id = row.Cells[0].Value
					t.klasse = row.Cells[1].Value
					t.naam = row.Cells[4].Value
					t.pd = row.Cells[8].Value
					t.vereniging = v
					v.teams[t.id] = t

					log.Printf("Team: Id: %s, Naam:%s, Klasse: %s", t.id, t.naam, t.klasse)

				}
			}
		}
	}

	return sb
}
