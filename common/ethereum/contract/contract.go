package contract

// https://eips.ethereum.org/EIPS/eip-20
//go:generate abigen --abi ./erc20/erc20.abi --pkg erc20 --type ERC20 --out ./erc20/erc20.go
// https://eips.ethereum.org/EIPS/eip-721
//go:generate abigen --abi ./erc721/erc721.abi --pkg erc721 --type ERC721 --out ./erc721/erc721.go
// https://eips.ethereum.org/EIPS/eip-1155
//go:generate abigen --abi ./erc1155/erc1155.abi --pkg erc1155 --type ERC1155 --out ./erc1155/erc1155.go
