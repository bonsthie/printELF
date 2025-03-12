package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"unsafe"
)

// ELF64 because who use 32 in big 2025 ...
type ELFHdr struct {
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

type ELFShdr struct {
	sh_name      uint32 // Section name (offset into the section header string table)
	sh_type      uint32 // Section type (SHT_PROGBITS, SHT_SYMTAB, etc.)
	sh_flags     uint64 // Section attributes (writable, executable, etc.)
	sh_addr      uint64 // Virtual address in memory where the section will be loaded
	sh_offset    uint64 // Offset of the section in the file
	sh_size      uint64 // Size of the section in bytes
	sh_link      uint32 // Link to another section (depends on type)
	sh_info      uint32 // Additional information (depends on type)
	sh_addralign uint64 // Alignment of the section
	sh_entsize   uint64 // Size of entries, if the section holds a table
}

type ELFSym struct {
	st_name  uint32 // Offset into the string table for the symbol name
	st_info  uint8  // Type and binding attributes of the symbol
	st_other uint8  // Visibility of the symbol
	st_shndx uint16 // Section index the symbol is in
	st_value uint64 // Symbol value (e.g., address)
	st_size  uint64 // Size of the symbol
}

func getSymbolName(offset uint32, strtab []byte) string {
	if offset >= uint32(len(strtab)) {
		return "<corrupt>"
	}

	end := bytes.IndexByte(strtab[offset:], 0)
	if end == -1 {
		return "<corrupt>"
	}

	return string(strtab[offset : offset+uint32(end)])
}

func getElfHeader(file *os.File) (*ELFHdr, error) {

	headerSize := int(unsafe.Sizeof(ELFHdr{}))
	buffer := make([]byte, headerSize)

	_, err := file.Read(buffer)
	if err != nil {
		return nil, err
	}
	return (*ELFHdr)(unsafe.Pointer(&buffer[0])), err
}

func getSectionHeader(file *os.File, header *ELFHdr, index uint16) (*ELFShdr, error) {
	offset := int64(header.Shoff) + int64(index)*int64(header.Shentsize)
	_, err := file.Seek(offset, io.SeekStart)
	if err != nil {
		return nil, err
	}

	buffer := make([]byte, int(unsafe.Sizeof(ELFShdr{})))
	_, err = file.Read(buffer)
	if err != nil {
		return nil, err
	}

	return (*ELFShdr)(unsafe.Pointer(&buffer[0])), nil
}

func getShstrtab(file *os.File, header *ELFHdr) ([]byte, error) {
	shstrtabHdr, err := getSectionHeader(file, header, header.Shstrndx)
	if err != nil {
		return nil, err
	}

	shstrtab := make([]byte, shstrtabHdr.sh_size)
	_, err = file.Seek(int64(shstrtabHdr.sh_offset), io.SeekStart)
	if err != nil {
		return nil, err
	}

	_, err = file.Read(shstrtab)
	if err != nil {
		return nil, err
	}

	return shstrtab, nil
}

func getSectionHeaderByName(file *os.File, header *ELFHdr, shstrtab []byte, name string) (*ELFShdr, error) {
	_, err := file.Seek(int64(header.Shoff), io.SeekStart)

	ShdrSize := int(unsafe.Sizeof(ELFShdr{}))
	buffer := make([]byte, ShdrSize)

	_, err = file.Seek(int64(header.Shoff), io.SeekStart)
	if err != nil {
		return nil, err
	}

	fmt.Printf("header.Shnum: %v\n", header.Shnum)
	for i := uint16(0); i < header.Shnum; i++ {
		_, err := file.Read(buffer)
		if err != nil {
			return nil, err
		}
		section := (*ELFShdr)(unsafe.Pointer(&buffer[0]))
		sectionNameStr := getSymbolName(section.sh_name, shstrtab)
		fmt.Printf("sectionName: %v\n", sectionNameStr)

		if sectionNameStr == name {
			return section, nil
		}
	}
	return nil, fmt.Errorf("section %v not found sadge :\\", name)

}

func readELFSection(file *os.File, section *ELFShdr) ([]byte, error) {
	_, err := file.Seek(int64(section.sh_offset), io.SeekStart)
	if err != nil {
		return nil, err
	}

	// Read section into memory
	data := make([]byte, section.sh_size)
	_, err = file.Read(data)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func listSymbols(file *os.File, symtab, strtab *ELFShdr) error {
	symtabData, err := readELFSection(file, symtab)
	if err != nil {
		return err
	}

	strtabData, err := readELFSection(file, strtab)
	if err != nil {
		return err
	}

	SymSize := uint64(unsafe.Sizeof(ELFSym{}))
	nbSym := symtab.sh_size / SymSize

	fmt.Println("Symbol Table Entries:")

	for i := uint64(0); i < nbSym; i++ {
		sym := (*ELFSym)(unsafe.Pointer(&symtabData[i*SymSize]))

		symNameOffset := sym.st_name
		symName := ""
		if symNameOffset < uint32(len(strtabData)) {
			symName = getSymbolName(symNameOffset, strtabData)
		}
		fmt.Printf("%016x %s\n", sym.st_value, symName)
	}

	return nil
}

// i didn't map the file and cast like i allways do so that lseek everywhere
func displaySymbol(file *os.File) error {

	header, err := getElfHeader(file)
	if err != nil {
		return fmt.Errorf("Error parsing header: %v", err)
	}

	Shstrtab, err := getShstrtab(file, header)
	if err != nil {
		return fmt.Errorf("Error retreving Shstrtab: %v", err)
	}

	symtab, err := getSectionHeaderByName(file, header, Shstrtab, ".symtab")
	if err != nil {
		return fmt.Errorf("Error searching Section: %v", err)
	}

	strtab, err := getSectionHeaderByName(file, header, Shstrtab, ".strtab")
	if err != nil {
		return fmt.Errorf("Error searching Section: %v", err)
	}

	return listSymbols(file, symtab, strtab)
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

	err = displaySymbol(file)
	if err != nil {
		fmt.Println("Error displaying symbol:", err)
	}

}
