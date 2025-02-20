package Models

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
)

type Request struct {
	Command string `json:"command"`
}

func ValidateInput(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("invalid command")
	}

	cmdType := strings.ToUpper(args[0])
	switch cmdType {

	case "SET":
		if err := validateSET(args); err != nil {
			return err
		}

	case "GET":
		if err := validateGET(args); err != nil {
			return err
		}

	case "QPUSH":
		if err := validateQueuePush(args); err != nil {
			return err
		}

	case "QPOP":
		if err := validateQueuePop(args); err != nil {
			return err
		}

	case "BQPOP":
		if err := validateBQPop(args); err != nil {
			return err
		}

	default:
		return fmt.Errorf("invalid command")
	}

	return nil
}

// There can be max of 6 args in SET -> checking each individually
func validateSET(args []string) error {
	if len(args) < 3 || len(args) > 6 {
		return fmt.Errorf("invalid SET command format. Usage: SET <key> <value> [EX <expiry>] [NX|XX]")
	}

	// checking for SET keyword
	arg0 := strings.ToUpper(args[0])
	if arg0 != "SET" {
		log.Printf("SET keyword error. user input: %s\n", arg0)
		return fmt.Errorf("invalid SET command format. Usage: SET <key> <value> [EX <expiry>] [NX|XX]")
	}

	condition := ""

	// Process optional arguments
	for i := 3; i < len(args); i++ {
		arg := strings.ToUpper(args[i])
		if arg == "EX" {
			if i+1 == len(args) {
				return fmt.Errorf("missing expiry time after EX")
			}
			// Convert expiry time to integer
			val, err := strconv.Atoi(args[i+1])
			if val < 0 {
				return errors.New("expiry time must be positive integer")
			}

			if err != nil {
				return errors.New("expiry time must be an integer")
			}
			i++
		} else if arg == "NX" || arg == "XX" {
			if condition != "" {
				return fmt.Errorf("either xx or nx can be used at a time.")
			}
			condition = arg
		} else {
			return fmt.Errorf("invalid argument: " + arg)
		}
	}

	return nil
}

func validateGET(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("invalid GET command format. Usage: GET <key>")
	}

	arg0 := strings.ToUpper(args[0])
	if arg0 != "GET" {
		return fmt.Errorf("invalid GET command format. Usage: GET <key>")
	}

	return nil
}

func validateQueuePush(args []string) error {
	if len(args) < 3 {
		return fmt.Errorf("invalid QPUSH command format. Usage: QPUSH <key> ...value")
	}

	return nil
}

func validateQueuePop(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("invalid QPOP command format. Usage: QPOP <key>")
	}

	return nil
}

func validateBQPop(args []string) error {
	if len(args) != 3 {
		return fmt.Errorf("invalid BQPOP command format. BQPOP <key> <timeout>")
	}

	val, err := strconv.Atoi(args[2])
	if err != nil {
		return fmt.Errorf("expiry time must be an integer")
	}

	if val < 0 {
		return fmt.Errorf("expiry time must be a positive integer")
	}

	return nil
}
