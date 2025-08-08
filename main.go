package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
        "math/rand"
        "time"
        "errors"
)

type config struct {
	Next     *string
	Previous *string
        AreaName *string
        PokemonCache map[string]LocationAreaResponse
        caughtPokemon map[string]PokemonData
        pokeapiClient *PokeapiClient
}

type PokeAPILocationAreasResponse struct {
	Count    int     `json:"count"`
	Next     *string `json:"next"`
	Previous *string `json:"previous"`
	Results  []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"results"`
}

type LocationAreaResponse struct {
    PokemonEncounters []PokemonEncounter `json:"pokemon_encounters"`
}

type PokemonEncounter struct {
    Pokemon Pokemon `json:"pokemon"`
}

type Pokemon struct {
    Name string `json:"name"`
    URL  string `json:"url"`
}

type cliCommand struct {
	name        string
	description string
	callback    func(args []string, cfg *config) error
}

type PokemonData struct {
    Name   string `json:"name"`
    Height int    `json:"height"`
    Weight int    `json:"weight"`
    Stats  []struct {
        Stat struct {
            Name string `json:"name"`
        } `json:"stat"`
        BaseStat int `json:"base_stat"`
    } `json:"stats"`
    Types []struct {
        Type struct {
            Name string `json:"name"`
        } `json:"type"`
    } `json:"types"`
    BaseExperience int `json:"base_experience"`
}

type PokeapiClient struct {
    httpClient *http.Client
}

func commandExit(args []string, cfg *config) error {
	fmt.Print("Closing the Pokedex... Goodbye!\n")
	os.Exit(0)
	return nil
}

func commandHelp(args []string, cfg *config) error {
	fmt.Print("Welcome to the Pokedex!\nUsage:\n\nhelp: Displays a help message\nexit: Exit the Pokedex\n")
	return nil
}

func commandMap(args []string, cfg *config) error {
	url := "https://pokeapi.co/api/v2/location-area/"

	if cfg.Next != nil {
		url = *cfg.Next
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var pokeResp PokeAPILocationAreasResponse

	err = json.Unmarshal(bodyBytes, &pokeResp)
	if err != nil {
		return err
	}

	cfg.Next = pokeResp.Next
	cfg.Previous = pokeResp.Previous

	for _, location := range pokeResp.Results {
		fmt.Println(location.Name)
	}

	return nil
}

func commandMapb(args []string, cfg *config) error {
	if cfg.Previous == nil {
		fmt.Println("you're on the first page")
		return nil
	}

	url := *cfg.Previous

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var pokeResp PokeAPILocationAreasResponse

	err = json.Unmarshal(bodyBytes, &pokeResp)
	if err != nil {
		return err
	}

	cfg.Next = pokeResp.Next
	cfg.Previous = pokeResp.Previous

	for _, location := range pokeResp.Results {
		fmt.Println(location.Name)
	}

	return nil
}

func commandExplore(args []string, cfg *config) error {
    if cfg.AreaName == nil {
        fmt.Println("Error: explore command requires an area name (e.g., explore pastoria-city-area)")
        return nil
    }

    areaName := *cfg.AreaName
    fmt.Printf("Exploring %s...\n", areaName)

    if cachedResponse, ok := cfg.PokemonCache[areaName]; ok {
        fmt.Println("Found Pokemon (from cache):")
        for _, encounter := range cachedResponse.PokemonEncounters {
            fmt.Printf(" - %s\n", encounter.Pokemon.Name)
        }
        return nil
    }

    url := "https://pokeapi.co/api/v2/location-area/" + areaName + "/"

    resp, err := http.Get(url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    bodyBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return err
    }

    var exploreResp LocationAreaResponse

    err = json.Unmarshal(bodyBytes, &exploreResp)
    if err != nil {
        return err
    }

    cfg.PokemonCache[areaName] = exploreResp

    fmt.Println("Found Pokemon:")
    for _, encounter := range exploreResp.PokemonEncounters {
        fmt.Printf(" - %s\n", encounter.Pokemon.Name)
    }

    return nil
}

func commandCatch(args []string, cfg *config) error {
	if len(args) != 1 {
		return errors.New("you must provide a pokemon name")
	}

	name := args[0]
	pokemon, err := cfg.pokeapiClient.GetPokemon(name)
	if err != nil {
		return err
	}

	fmt.Printf("Throwing a Pokeball at %s...\n", pokemon.Name)
        catchChance := rand.Intn(100)
        escapeThreshold := 100 - (pokemon.BaseExperience / 4)
        if escapeThreshold < 5 {
            escapeThreshold = 5
        }
        if catchChance > escapeThreshold {
            fmt.Printf("%s escaped!\n", pokemon.Name)
            return nil
        }

	fmt.Printf("%s was caught!\n", pokemon.Name)
        fmt.Println("You may now inspect it with the inspect command.")

	cfg.caughtPokemon[pokemon.Name] = pokemon
	return nil
}

func (c *PokeapiClient) GetPokemon(pokemonName string) (PokemonData, error) {
    url := "https://pokeapi.co/api/v2/pokemon/" + pokemonName
    resp, err := c.httpClient.Get(url)
    if err != nil {
        return PokemonData{}, fmt.Errorf("error fetching Pokemon data: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return PokemonData{}, fmt.Errorf("API returned non-OK status: %s", resp.Status)
    }

    bodyData, err := io.ReadAll(resp.Body)
    if err != nil {
        return PokemonData{}, fmt.Errorf("error reading response body: %w", err)
    }

    var pokemonInfo PokemonData
    err = json.Unmarshal(bodyData, &pokemonInfo)
    if err != nil {
        return PokemonData{}, fmt.Errorf("error unmarshaling Pokemon data: %w", err)
    }
    return pokemonInfo, nil
}

func commandInspect(args []string, cfg *config) error {
    caught, ok := cfg.caughtPokemon[args[0]]
    if !ok {
        fmt.Println("you have not caught that pokemon")
        return nil
    }

    fmt.Println("Name:", caught.Name)
    fmt.Println("Height:", caught.Height)
    fmt.Println("Weight:", caught.Weight)

    fmt.Println("Stats:")
    for _, s := range caught.Stats {
        fmt.Printf("  -%s: %d\n", s.Stat.Name, s.BaseStat)
    }

    fmt.Println("Types:")
    for _, t := range caught.Types {
        fmt.Printf("  - %s\n", t.Type.Name)
    }

    return nil
}

func commandPokedex(args []string, cfg *config) error {
	fmt.Println("Your Pokedex:")
	for _, p := range cfg.caughtPokemon {
		fmt.Printf(" - %s\n", p.Name)
	}
	return nil
}

func main() {
        rand.Seed(time.Now().UnixNano())
	cfg := &config{
           PokemonCache: make(map[string]LocationAreaResponse),
           caughtPokemon:  make(map[string]PokemonData),
           pokeapiClient: &PokeapiClient{
               httpClient: &http.Client{},
           },
        }

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("Pokedex > ")
		scanner.Scan()
		input := scanner.Text()
		cleanedInput := strings.ToLower(input)
		words := strings.Fields(cleanedInput)
		if len(words) == 0 {
			continue
		}
		firstWord := words[0]
		commands := map[string]cliCommand{
			"exit": {
				name:        "exit",
				description: "Exit the Pokedex",
				callback:    commandExit,
			},
			"help": {
				name:        "help",
				description: "Displays a help message",
				callback:    commandHelp,
			},
			"map": {
				name:        "map",
				description: "Display location areas",
				callback:    commandMap,
			},
			"mapb": {
				name:        "mapb",
				description: "Display previous location areas",
				callback:    commandMapb,
			},
			"explore": {
				name:        "explore",
				description: "Explore certain location areas",
				callback:    commandExplore,
			},
                        "catch": {
                            name:        "catch",
                            description: "Try to catch a pokemon by name",
                            callback:    commandCatch,
                        },
                        "inspect": {
                            name:        "inspect",
                            description: "Inspect a caught pokemon's details",
                            callback:    commandInspect,
                        },
                   	"pokedex": {
			name:        "pokedex",
			description: "See all the pokemon you've caught",
			callback:    commandPokedex,
		        },
		}
		if command, exists := commands[firstWord]; exists {
			if firstWord == "explore" {
				if len(words) < 2 {
					fmt.Println("Error: explore command requires an area name")
					continue
				}
				cfg.AreaName = &words[1]
			}

			err := command.callback(words[1:], cfg)
			if err != nil {
				fmt.Println(err)
			}
		} else {
			fmt.Println("Unknown command")
		}
	}
}
