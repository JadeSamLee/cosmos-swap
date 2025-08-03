import { SigningCosmWasmClient } from "@cosmjs/cosmwasm-stargate";
import { DirectSecp256k1Wallet } from "@cosmjs/proto-signing";
import { GasPrice } from "@cosmjs/stargate";

// This is a placeholder for your deployed contract address and RPC endpoint
const RPC_ENDPOINT = "http://localhost:26657"; // Example local CosmWasm node
const CONTRACT_ADDRESS = "cosmos1...your_contract_address..."; // Replace with your deployed contract address

async function main() {
  // Generate a random wallet for demonstration purposes
  const wallet = await DirectSecp256k1Wallet.generate();
  const [firstAccount] = await wallet.getAccounts();

  // Connect to the CosmWasm client
  const gasPrice = GasPrice.fromString("0.002uatom"); // Example gas price
  const client = await SigningCosmWasmClient.connectWithSigner(
    RPC_ENDPOINT,
    wallet,
    { gasPrice: gasPrice }
  );

  console.log("\n--- CosmWasm Contract Interaction ---");
  console.log(`Connected to: ${RPC_ENDPOINT}`);
  console.log(`Wallet address: ${firstAccount.address}`);

  // Example: Querying the contract (replace with your contract's query message)
  try {
    const queryResponse = await client.queryContractSmart(CONTRACT_ADDRESS, { get_count: {} });
    console.log("Query response:", queryResponse);
  } catch (error) {
    console.error("Error querying contract:", error);
  }

  // Example: Executing a contract message (replace with your contract's execute message)
  try {
    const executeResponse = await client.execute(
      firstAccount.address,
      CONTRACT_ADDRESS,
      { increment: {} }, // Example execute message
      "auto",
      "Increment counter",
      [{ denom: "uatom", amount: "1000" }] // Example funds
    );
    console.log("Execute response:", executeResponse);
  } catch (error) {
    console.error("Error executing contract:", error);
  }
}

main().catch(console.error);

