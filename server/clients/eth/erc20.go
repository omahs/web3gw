package goethclient

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/threefoldtech/web3_proxy/server/clients/eth/erc20"
)

func (c *Client) GetTokenBalance(contractAddress string) (*big.Int, error) {
	token, err := erc20.NewErc20(common.HexToAddress(contractAddress), c.Eth)
	if err != nil {
		return nil, err
	}

	return token.BalanceOf(&bind.CallOpts{}, c.Address)
}

func (c *Client) TransferTokens(ctx context.Context, contractAddress common.Address, target string, amount int64) (string, error) {
	token, err := erc20.NewErc20(contractAddress, c.Eth)
	if err != nil {
		return "", err
	}

	opts, err := c.getDefaultTransactionOpts(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to get default transaction opts")
	}

	tx, err := token.Transfer(opts, common.HexToAddress(target), big.NewInt(amount))
	if err != nil {
		return "", err
	}

	r, err := bind.WaitMined(ctx, c.Eth, tx)
	if err != nil {
		log.Err(err).Msg("failed to wait for tft approval")
		return "", err
	}

	log.Debug().Msgf("Approve spend tx mined: %s, block %d, gas: %d, status: %d", tx.Hash().Hex(), r.BlockNumber, r.GasUsed, r.Status)

	return tx.Hash().Hex(), nil
}

func (c *Client) ApproveTokenSpending(ctx context.Context, contractAddress, spender string, amount int64) (string, error) {
	token, err := erc20.NewErc20(common.HexToAddress(contractAddress), c.Eth)
	if err != nil {
		return "", err
	}

	opts, err := c.getDefaultTransactionOpts(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to get default transaction opts")
	}

	tx, err := token.Approve(opts, common.HexToAddress(spender), big.NewInt(amount))
	if err != nil {
		return "", err
	}

	r, err := bind.WaitMined(ctx, c.Eth, tx)
	if err != nil {
		log.Err(err).Msg("failed to wait for tft approval")
		return "", err
	}

	log.Debug().Msgf("Approve spend tx mined: %s, block %d, gas: %d, status: %d", tx.Hash().Hex(), r.BlockNumber, r.GasUsed, r.Status)

	return tx.Hash().Hex(), nil
}

func (c *Client) TransferFromTokens(ctx context.Context, contractAddress, from, to string, amount int64) (string, error) {
	token, err := erc20.NewErc20(common.HexToAddress(contractAddress), c.Eth)
	if err != nil {
		return "", err
	}

	opts, err := c.getDefaultTransactionOpts(ctx)
	if err != nil {
		return "", errors.Wrap(err, "failed to get default transaction opts")
	}

	tx, err := token.TransferFrom(opts, common.HexToAddress(from), common.HexToAddress(to), big.NewInt(amount))
	if err != nil {
		return "", err
	}

	r, err := bind.WaitMined(ctx, c.Eth, tx)
	if err != nil {
		log.Err(err).Msg("failed to wait for tft approval")
		return "", err
	}

	log.Debug().Msgf("Approve spend tx mined: %s, block %d, gas: %d, status: %d", tx.Hash().Hex(), r.BlockNumber, r.GasUsed, r.Status)

	return tx.Hash().Hex(), nil
}
