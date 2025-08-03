import { ethers } from "ethers";

// This is a placeholder for your deployed contract ABI and address
const CONTRACT_ABI = [
  "function greet() view returns (string)",
  "function setGreeting(string _greeting)"
];
const CONTRACT_ADDRESS = "0x5FbDB2315678afecb367f032d93F642f64180aa3"; // Example address

async function main() {
  // Connect to an Ethereum provider (e.g., Infura, Alchemy, or a local node)
  const provider = new ethers.JsonRpcProvider("http://localhost:8545"); // Example local node

  // Get the signer (your account) from the provider
  const signer = await provider.getSigner();

  // Create a contract instance
  const contract = new ethers.Contract(CONTRACT_ADDRESS, CONTRACT_ABI, signer);

  console.log("\n--- EVM Contract Interaction ---");

  // Read from the contract
  try {
    const currentGreeting = await contract.greet();
    console.log(`Current greeting: ${currentGreeting}`);
  } catch (error) {
    console.error("Error reading greeting:", error);
  }

  // Write to the contract
  try {
    const newGreeting = "Hello from TypeScript!";
    const tx = await contract.setGreeting(newGreeting);
    console.log(`Setting new greeting: '${newGreeting}' (Transaction hash: ${tx.hash})`);
    await tx.wait();
    console.log("Transaction confirmed.");

    const updatedGreeting = await contract.greet();
    console.log(`Updated greeting: ${updatedGreeting}`);
  } catch (error) {
    console.error("Error setting greeting:", error);
  }
}

main().catch(console.error);

