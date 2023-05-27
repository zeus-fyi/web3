package web3_actions

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/rs/zerolog/log"
	"github.com/zeus-fyi/gochain/web3/accounts"
	web3_types "github.com/zeus-fyi/gochain/web3/types"
)

// CallConstantFunction executes a contract function call without submitting a transaction.
func (w *Web3Actions) CallConstantFunction(ctx context.Context, payload *SendContractTxPayload) ([]interface{}, error) {
	w.Dial()
	defer w.C.Close()
	if payload.SmartContractAddr == "" {
		err := errors.New("no contract address specified")
		log.Ctx(ctx).Err(err).Msg("CallConstantFunction")
		return nil, err
	}
	if payload.ContractFile == "erc20" {
		abiLoaded, err := web3_types.ABIBuiltIn(ERC20)
		if err != nil {
			return nil, err
		}
		payload.ContractABI = abiLoaded
	}
	if payload.ContractABI == nil {
		return nil, errors.New("no contract abi specified")
	}
	fn := payload.ContractABI.Methods[payload.MethodName]
	goParams, err := web3_types.ConvertArguments(fn.Inputs, payload.Params)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("CallConstantFunction: ConvertArguments")
		return nil, err
	}
	input, err := payload.ContractABI.Pack(payload.MethodName, goParams...)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("CallConstantFunction: myabi.Pack")
		return nil, fmt.Errorf("failed to pack values: %v", err)
	}
	scAddr := accounts.HexToAddress(payload.SmartContractAddr)

	var res []byte
	err = w.C.Client().Call(&res, payload.MethodName, web3_types.CallMsg{Data: input, To: &scAddr})
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("CallConstantFunction: client.Call")
		return nil, err
	}
	// TODO: calling a function on a contract errors on unpacking, it should probably know it's not a contract before hand if it can
	// fmt.Printf("RESPONSE: %v\n", string(res))
	vals, err := fn.Outputs.UnpackValues(res)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("CallConstantFunction: UnpackValues")
		return nil, fmt.Errorf("failed to unpack values from %s: %v", hexutil.Encode(res), err)
	}
	return convertOutputParams(vals), nil
}

func (w *Web3Actions) GetContractDecimals(ctx context.Context, contractAddress string) (int32, error) {
	payload := SendContractTxPayload{
		SmartContractAddr: contractAddress,
		ContractFile:      ERC20,
		SendEtherPayload:  SendEtherPayload{},
		MethodName:        Decimals,
		Params:            nil,
	}
	decimals, derr := w.GetContractConst(ctx, &payload)
	if derr != nil {
		log.Ctx(ctx).Err(derr).Msg("Web3Actions: GetContractDecimals")
		return 0, derr
	}
	dLen := len(decimals)
	if dLen != 1 {
		err := errors.New("contract call has unexpected return slice size")
		log.Ctx(ctx).Err(err).Interface("decimals", decimals).Msgf("Web3Actions: GetContractDecimals slice len: %d", dLen)
		return 0, derr
	}
	contractDecimals := int32(decimals[0].(uint8))
	return contractDecimals, derr
}