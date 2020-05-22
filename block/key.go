package block

import "fmt"

// Enter blocks until Enter is pressed
func Enter() {
	fmt.Println("Press Enter to exit..")
	_, _ = fmt.Scanln()
}
