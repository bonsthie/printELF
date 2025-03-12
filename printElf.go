
// ELF64 because who use 32 in big 2025 ...
type ELFHeader struct {
	Ident     [16]byte // Magic number and other info
	Type      uint16   // Object file type
	Machine   uint16   // Architecture
	Version   uint32   // ELF version
	Entry     uint64   // Entry point address
	Phoff     uint64   // Program header table file offset
	Shoff     uint64   // Section header table file offset
	Flags     uint32   // Processor-specific flags
	Ehsize    uint16   // ELF header size
	Phentsize uint16   // Program header table entry size
	Phnum     uint16   // Number of program header entries
	Shentsize uint16   // Section header table entry size
	Shnum     uint16   // Number of section header entries
	Shstrndx  uint16   // Section header string table index
}

func getElfHeader(file : int) {

}


func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <ELF file>")
		os.Exit(1)
	}

	filePath := os.Args[1]

	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file:", err)
		os.Exit(1)
	}
	defer file.Close()

}
