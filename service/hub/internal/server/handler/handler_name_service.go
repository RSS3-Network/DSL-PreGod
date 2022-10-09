package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/labstack/echo/v4"
	"github.com/naturalselectionlabs/pregod/common/datasource/ethereum"
	"github.com/naturalselectionlabs/pregod/common/datasource/ethereum/contract/crossbell"
	"github.com/naturalselectionlabs/pregod/common/datasource/ethereum/contract/lens"
	lenscontract "github.com/naturalselectionlabs/pregod/common/datasource/ethereum/contract/lens/contract"
	"github.com/naturalselectionlabs/pregod/common/datasource/ethereum/contract/spaceid"
	spaceidcontract "github.com/naturalselectionlabs/pregod/common/datasource/ethereum/contract/spaceid/contracts"
	"github.com/naturalselectionlabs/pregod/common/ethclientx"
	"github.com/naturalselectionlabs/pregod/common/protocol"
	"github.com/naturalselectionlabs/pregod/common/utils/httpx"
	"github.com/naturalselectionlabs/pregod/common/worker/ens"
	"github.com/naturalselectionlabs/pregod/service/hub/internal/server/model"
	"github.com/unstoppabledomains/resolution-go/v2"
	"github.com/unstoppabledomains/resolution-go/v2/namingservice"
	goens "github.com/wealdtech/go-ens/v3"
	"go.opentelemetry.io/otel"
)

func (h *Handler) GetNameResolveFunc(c echo.Context) error {
	go h.apiReport(GetNS, c)
	tracer := otel.Tracer("GetNameResolveFunc")
	_, httpSnap := tracer.Start(c.Request().Context(), "http")

	defer httpSnap.End()

	request := GetRequest{}

	if err := c.Bind(&request); err != nil {
		return BadRequest(c)
	}

	if err := c.Validate(&request); err != nil {
		return err
	}

	if len(request.Address) == 0 {
		return AddressIsEmpty(c)
	}

	go h.filterReport(GetNS, request)

	result := ReverseResolveAll(strings.ToLower(request.Address), true)

	if len(result.Address) == 0 {
		return AddressIsInvalid(c)
	}
	return c.JSON(http.StatusOK, result)
}

// LensResolveFunc temporary function to resolve Lens for Pinata
func (h *Handler) LensResolveFunc(c echo.Context) error {
	result, err := ResolveLens(c.Param("address"))
	if err != nil {
		return err
	}

	return c.JSON(http.StatusOK, result)
}

func ResolveCrossbell(input string) (string, error) {
	var result string
	ethereumClient, err := ethclientx.Global(protocol.NetworkCrossbell)
	if err != nil {
		return "", fmt.Errorf("failed to connect to crossbell rpc: %s", err)
	}

	characterContract, err := crossbell.NewCharacter(crossbell.AddressCharacter, ethereumClient)
	if err != nil {
		return "", fmt.Errorf("failed to connect to crossbell character contracts: %s", err)
	}

	if strings.HasSuffix(input, ".csb") {
		character, err := characterContract.GetCharacterByHandle(&bind.CallOpts{}, strings.TrimSuffix(input, ".csb"))
		if err != nil {
			return "", fmt.Errorf("failed to get crossbell character by handle: %s", err)
		}

		characterOwner, err := characterContract.OwnerOf(&bind.CallOpts{}, character.CharacterId)
		if err != nil {
			return "", fmt.Errorf("failed to get crossbell character owner: %s", err)
		}
		result = characterOwner.String()
	} else {
		characterId, err := characterContract.GetPrimaryCharacterId(&bind.CallOpts{}, common.HexToAddress(input))
		if err != nil {
			return "", fmt.Errorf("failed to get crossbell character by address: %s", err)
		}

		result, err = characterContract.GetHandle(&bind.CallOpts{}, characterId)
		if err != nil {
			return "", fmt.Errorf("failed to get crossbell handle by characterId: %s", err)
		}

		if result != "" && !strings.HasSuffix(result, ".csb") {
			result += ".csb"
		}
	}

	return strings.ToLower(result), nil
}

func ResolveENS(address string) (string, error) {
	result, err := ens.Resolve(address)
	if err != nil {
		return "", fmt.Errorf("failed to resolve ENS for: %s", address)
	}

	return strings.ToLower(result), nil
}

func ResolveLens(input string) (string, error) {
	var result string
	ethereumClient, err := ethclientx.Global(protocol.NetworkPolygon)
	if err != nil {
		return "", fmt.Errorf("failed to connect to polygon rpc: %s", err)
	}

	lensHubContract, err := lenscontract.NewHub(lens.HubProxyContractAddress, ethereumClient)
	if err != nil {
		return "", fmt.Errorf("failed to connect to lens hub contracts: %s", err)
	}

	if strings.HasSuffix(input, ".lens") {
		profileID, err := lensHubContract.GetProfileIdByHandle(&bind.CallOpts{}, input)
		if err != nil {
			return "", fmt.Errorf("failed to get lens profile id by handle: %s", err)
		}

		profileOwner, err := lensHubContract.OwnerOf(&bind.CallOpts{}, profileID)
		if err != nil {
			return "", fmt.Errorf("failed to get lens profile owner: %s", err)
		}

		result = profileOwner.String()
	} else {
		profileID, err := lensHubContract.DefaultProfile(&bind.CallOpts{}, common.HexToAddress(input))
		if err != nil {
			return "", fmt.Errorf("failed to get default lens profile id by address: %s", err)
		}

		result, _ = lensHubContract.GetHandle(&bind.CallOpts{}, profileID)
	}

	return strings.ToLower(result), nil
}

func ResolveSpaceID(input string) (string, error) {
	var result string
	ethereumClient, err := ethclientx.Global(protocol.NetworkBinanceSmartChain)
	if err != nil {
		return "", fmt.Errorf("failed to connect to bsc rpc: %s", err)
	}

	spaceidContract, err := spaceidcontract.NewSpaceid(spaceid.AddressSpaceID, ethereumClient)
	if err != nil {
		return "", fmt.Errorf("failed to new a space id contract: %w", err)
	}

	if strings.HasSuffix(input, ".bnb") {
		namehash, _ := goens.NameHash(input)

		resolver, err := spaceidContract.Resolver(&bind.CallOpts{}, namehash)
		if err != nil {
			return "", fmt.Errorf("failed to get space id resolver: %w", err)
		}

		if resolver == ethereum.AddressGenesis {
			return "", fmt.Errorf("the space id does not have a resolver: %s", input)
		}

		spaceidResolver, err := spaceidcontract.NewResolver(resolver, ethereumClient)
		if err != nil {
			return "", fmt.Errorf("failed to new to space id resolver contract: %w", err)
		}

		profileID, err := spaceidResolver.Addr(&bind.CallOpts{}, namehash)
		if err != nil {
			return "", fmt.Errorf("failed to get Space ID by handle: %s", err)
		}

		result = profileID.String()
	} else {
		namehash, _ := goens.NameHash(fmt.Sprintf("%s.addr.reverse", strings.TrimPrefix(input, "0x")))

		resolver, err := spaceidContract.Resolver(&bind.CallOpts{}, namehash)
		if err != nil {
			return "", fmt.Errorf("failed to get space id resolver: %w", err)
		}

		if resolver == ethereum.AddressGenesis {
			return "", fmt.Errorf("the space id does not have a resolver: %s", input)
		}

		spaceidResolver, err := spaceidcontract.NewResolver(resolver, ethereumClient)
		if err != nil {
			return "", fmt.Errorf("failed to new to space id resolver contract: %w", err)
		}

		if result, err = spaceidResolver.Name(&bind.CallOpts{}, namehash); err != nil {
			return "", fmt.Errorf("failed to get space id handle by address: %w", err)
		}
	}

	return strings.ToLower(result), nil
}

func ResolveUnstoppableDomains(input string) (string, error) {
	if strings.Contains(input, ".") {
		unsBuilder := resolution.NewUnsBuilder()
		ethClient, err := ethclientx.Global(protocol.NetworkEthereum)
		if err != nil {
			return "", fmt.Errorf("failed to connect to ethereum rpc: %s", err)
		}

		polygonClient, err := ethclientx.Global(protocol.NetworkPolygon)
		if err != nil {
			return "", fmt.Errorf("failed to connect to polygon rpc: %s", err)
		}

		unsBuilder.SetContractBackend(ethClient)
		unsBuilder.SetL2ContractBackend(polygonClient)

		unsResolution, err := unsBuilder.Build()
		if err != nil {
			return "", fmt.Errorf("failed to build unsResolution: %s", err)
		}

		znsResolution, err := resolution.NewZnsBuilder().Build()
		if err != nil {
			return "", fmt.Errorf("failed to build znsResolution: %s", err)
		}

		namingServices := map[string]resolution.NamingService{namingservice.UNS: unsResolution, namingservice.ZNS: znsResolution}
		namingServiceName, _ := resolution.DetectNamingService(input)
		if namingServices[namingServiceName] != nil {
			resolvedAddress, err := namingServices[namingServiceName].Addr(input, "ETH")
			if err != nil {
				return "", fmt.Errorf("failed to result %s: %s", namingServiceName, err)
			}
			return strings.ToLower(resolvedAddress), nil
		}

	}

	return "", nil
}

func ResolveBit(input string) (string, error) {
	bitResult := &model.BitResult{}
	bitEndpoint := "indexer-v1.did.id"
	request := http.Request{Method: http.MethodPost, URL: &url.URL{Scheme: "https", Host: bitEndpoint, Path: "/"}}

	if strings.HasSuffix(input, ".bit") {
		request.Body = io.NopCloser(strings.NewReader(fmt.Sprintf(`{
			"jsonrpc": "2.0",
			"id": 1,
			"method": "das_accountInfo",
			"params": [
				{
					"account": "%s"
				}
			]
		}`, input)))
	} else {
		request.Body = io.NopCloser(strings.NewReader(fmt.Sprintf(`{
			"jsonrpc": "2.0",
			"id": 1,
			"method": "das_reverseRecord",
			"params": [
				{
					"type": "blockchain",
					"key_info": {
						"coin_type": "60",
						"chain_id": "1",
						"key": "%s"
					}
				}
			]
		}`, input)))
	}

	err := httpx.DoRequest(context.Background(), http.DefaultClient, &request, &bitResult)

	defer request.Body.Close()

	if err != nil {
		return "", fmt.Errorf("failed to request .bit resolver: %s", err)
	}

	if bitResult.Result.Error != "" {
		return "", fmt.Errorf(".bit resolver returned an error: %s", bitResult.Result.Error)
	}

	if strings.HasSuffix(input, ".bit") {
		return bitResult.Result.Data.AccountInfo.Address, nil
	} else {
		return bitResult.Result.Data.Account, nil
	}
}

func ResolveAll(result *model.NameServiceResult) {
	if result.ENS == "" {
		result.ENS, _ = ResolveENS(result.Address)
	}

	if result.Crossbell == "" {
		result.Crossbell, _ = ResolveCrossbell(result.Address)
	}

	if result.Lens == "" {
		result.Lens, _ = ResolveLens(result.Address)
	}

	if result.SpaceID == "" {
		result.SpaceID, _ = ResolveSpaceID(result.Address)
	}

	if result.UnstoppableDomains == "" {
		result.UnstoppableDomains, _ = ResolveUnstoppableDomains(result.Address)
	}

	if result.Bit == "" {
		result.Bit, _ = ResolveBit(result.Address)
	}
}

func ReverseResolveAll(input string, resolveAll bool) model.NameServiceResult {
	result := model.NameServiceResult{}
	splits := strings.Split(input, ".")
	var address string

	switch splits[len(splits)-1] {
	case "eth":
		// error here means the address doesn't have a primary ENS, and can be ignored
		address, _ = ResolveENS(input)
		result.ENS = input
	case "csb":
		address, _ = ResolveCrossbell(input)
		result.Crossbell = input
	case "lens":
		address, _ = ResolveLens(input)
		result.Lens = input
	case "bnb":
		address, _ = ResolveSpaceID(input)
		result.SpaceID = input
	case "crypto", "nft", "blockchain", "bitcoin", "coin", "wallet", "888", "dao", "x", "zil":
		address, _ = ResolveUnstoppableDomains(input)
		result.UnstoppableDomains = input
	case "bit":
		address, _ = ResolveBit(input)
		result.Bit = input
	default:
		if ValidateEthereumAddress(input) {
			address = input
		}
	}
	if address != "" {
		result.Address = address
		if resolveAll {
			ResolveAll(&result)
		}
	}

	return result
}

func ValidateEthereumAddress(address string) bool {
	if address == "" {
		return false
	}

	re := regexp.MustCompile("^0x[0-9a-fA-F]{40}$")
	return re.MatchString(address)
}
