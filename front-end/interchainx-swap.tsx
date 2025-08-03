"use client"

import { useState, useEffect } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Card, CardContent } from "@/components/ui/card"
import { Dialog, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { ArrowUpDown, Wallet, Check, Clock, Eye, LogOut, RefreshCw } from "lucide-react"
import { ethers } from "ethers"
import Image from "next/image"

interface SwapStep {
  id: number
  title: string
  completed: boolean
  current: boolean
}

export default function Component() {
  const [fromAmount, setFromAmount] = useState("0.0")
  const [toAmount, setToAmount] = useState("0.0")
  const [fromToken, setFromToken] = useState("ETH")
  const [toToken, setToToken] = useState("TCRO")
  const [isConnected, setIsConnected] = useState(false)
  const [account, setAccount] = useState("")
  const [network, setNetwork] = useState("")
  const [showSwapModal, setShowSwapModal] = useState(false)
  const [secret, setSecret] = useState("")
  const [swapStarted, setSwapStarted] = useState(false)
  const [showRevealButton, setShowRevealButton] = useState(false)
  const [swapCompleted, setSwapCompleted] = useState(false)
  const [currentStep, setCurrentStep] = useState(0)
  const [isExecutingContract, setIsExecutingContract] = useState(false)
  const [isConnecting, setIsConnecting] = useState(false)
  const [showClaimButton, setShowClaimButton] = useState(false)

  const ethToTcroSteps: SwapStep[] = [
    { id: 1, title: "Order created", completed: false, current: false },
    { id: 2, title: "Cronos HTLC created", completed: false, current: false },
    { id: 3, title: "EVM Escrow created", completed: false, current: false },
    { id: 4, title: "Cronos HTLC funded", completed: false, current: false },
    { id: 5, title: "Cross-chain transfer initiated", completed: false, current: false },
    { id: 6, title: "Maker claimed Cronos", completed: false, current: false },
    { id: 7, title: "Taker claimed Eth", completed: false, current: false },
  ]

  const tcroToEthSteps: SwapStep[] = [
    { id: 1, title: "Order created", completed: false, current: false },
    { id: 2, title: "HTLC created", completed: false, current: false },
    { id: 3, title: "Escrow created", completed: false, current: false },
    { id: 4, title: "HTLC funded", completed: false, current: false },
    { id: 5, title: "Cross-chain transfer initiated", completed: false, current: false },
    { id: 6, title: "Maker claimed Eth", completed: false, current: false },
    { id: 7, title: "Taker claimed TCRO", completed: false, current: false },
  ]

  const [steps, setSteps] = useState<SwapStep[]>(ethToTcroSteps)

  const sepoliaNetwork = {
    chainId: "0xaa36a7",
    chainName: "Sepolia test network",
    rpcUrls: ["https://sepolia.infura.io/v3/"],
    nativeCurrency: {
      name: "ETH",
      symbol: "ETH",
      decimals: 18,
    },
    blockExplorerUrls: ["https://sepolia.etherscan.io/"],
  }

  const cronosNetwork = {
    chainId: "0x152",
    chainName: "Cronos Testnet",
    rpcUrls: ["https://evm-t3.cronos.org/"],
    nativeCurrency: {
      name: "TCRO",
      symbol: "TCRO",
      decimals: 18,
    },
    blockExplorerUrls: ["https://testnet.cronoscan.com/"],
  }

  // Clean disconnect without page refresh
  const cleanDisconnect = () => {
    setIsConnected(false)
    setAccount("")
    setNetwork("")
    setFromAmount("0.0")
    setToAmount("0.0")
    setSecret("")
    setSwapStarted(false)
    setShowRevealButton(false)
    setShowClaimButton(false)
    setSwapCompleted(false)
    setCurrentStep(0)
    setShowSwapModal(false)
    setIsConnecting(false)

    // Remove event listeners
    if (typeof window.ethereum !== "undefined") {
      try {
        window.ethereum.removeAllListeners("accountsChanged")
        window.ethereum.removeAllListeners("chainChanged")
        window.ethereum.removeAllListeners("connect")
        window.ethereum.removeAllListeners("disconnect")
      } catch (error) {
        console.log("Error removing listeners:", error)
      }
    }
  }

  // Full page refresh (only for manual refresh button)
  const fullRefresh = () => {
    window.location.reload()
  }

  const connectWallet = async () => {
    if (typeof window.ethereum !== "undefined") {
      try {
        setIsConnecting(true)

        // Clean any existing state first (without page refresh)
        cleanDisconnect()

        // Request fresh connection
        const accounts = await window.ethereum.request({
          method: "eth_requestAccounts",
        })

        if (accounts.length > 0) {
          setAccount(accounts[0])
          setIsConnected(true)

          const chainId = await window.ethereum.request({ method: "eth_chainId" })
          updateNetworkStatus(chainId)

          // Set up fresh event listeners
          window.ethereum.on("accountsChanged", handleAccountsChanged)
          window.ethereum.on("chainChanged", handleChainChanged)
          window.ethereum.on("disconnect", handleDisconnect)

          console.log("Wallet connected successfully:", accounts[0])
        }

        setIsConnecting(false)
      } catch (error) {
        console.error("Error connecting wallet:", error)
        setIsConnecting(false)
        cleanDisconnect()
      }
    } else {
      alert("MetaMask is not installed!")
      setIsConnecting(false)
    }
  }

  const disconnectWallet = () => {
    cleanDisconnect()
  }

  const handleAccountsChanged = (accounts: string[]) => {
    console.log("Accounts changed:", accounts)
    if (accounts.length === 0) {
      // No accounts connected
      cleanDisconnect()
    } else if (accounts[0] !== account) {
      // Account changed, update to new account
      setAccount(accounts[0])
      console.log("Account updated to:", accounts[0])
    }
  }

  const handleChainChanged = (chainId: string) => {
    console.log("Chain changed:", chainId)
    updateNetworkStatus(chainId)
    // Don't disconnect, just update network status
  }

  const handleDisconnect = () => {
    console.log("Wallet disconnected")
    cleanDisconnect()
  }

  const updateNetworkStatus = (chainId: string) => {
    if (chainId === sepoliaNetwork.chainId) {
      setNetwork("Sepolia")
    } else if (chainId === cronosNetwork.chainId) {
      setNetwork("Cronos")
    } else {
      setNetwork("Unknown")
    }
  }

  const switchNetwork = async (targetNetwork: "sepolia" | "cronos") => {
    if (typeof window.ethereum !== "undefined") {
      try {
        const networkConfig = targetNetwork === "sepolia" ? sepoliaNetwork : cronosNetwork

        await window.ethereum.request({
          method: "wallet_switchEthereumChain",
          params: [{ chainId: networkConfig.chainId }],
        })

        // Network will be updated via handleChainChanged
        console.log(`Switched to ${targetNetwork}`)
      } catch (error: any) {
        if (error.code === 4902) {
          try {
            const networkConfig = targetNetwork === "sepolia" ? sepoliaNetwork : cronosNetwork
            await window.ethereum.request({
              method: "wallet_addEthereumChain",
              params: [networkConfig],
            })
            console.log(`Added and switched to ${targetNetwork}`)
          } catch (addError) {
            console.error("Error adding network:", addError)
          }
        } else {
          console.error("Error switching network:", error)
        }
      }
    }
  }

  const executeCrossChainTransfer = async (isEthToTcro: boolean) => {
    try {
      const provider = new ethers.BrowserProvider(window.ethereum)
      const signer = await provider.getSigner()

      if (isEthToTcro) {
        // Step 1: Send TCRO from 0x35fDA6152CEd2c76f27E3828d1Cb691208EEFf65 to 0x83F0f43dF0836e9fD50517E3c0897fB9c5b35657
        console.log("Initiating TCRO transfer...")

        // Step 2: Send equivalent ETH from 0x83F0f43dF0836e9fD50517E3c0897fB9c5b35657 to 0x35fDA6152CEd2c76f27E3828d1Cb691208EEFf65
        console.log("Initiating ETH transfer...")

        // For demo purposes, we'll send a transaction from the connected wallet
        const tx = await signer.sendTransaction({
          to: "0x35fDA6152CEd2c76f27E3828d1Cb691208EEFf65",
          value: ethers.parseEther("0.001"), // 0.001 ETH
          gasLimit: 100000,
        })

        console.log("Cross-chain transfer initiated:", tx.hash)
        await tx.wait()
        console.log("Cross-chain transfer completed")

        return true
      } else {
        // TCRO to ETH swap
        console.log("Initiating ETH transfer...")

        // Send ETH from 0x83F0f43dF0836e9fD50517E3c0897fB9c5b35657 to 0x35fDA6152CEd2c76f27E3828d1Cb691208EEFf65
        const tx = await signer.sendTransaction({
          to: "0x35fDA6152CEd2c76f27E3828d1Cb691208EEFf65",
          value: ethers.parseEther("42.02"), // 42.02 TCRO equivalent
          gasLimit: 100000,
        })

        console.log("Cross-chain transfer initiated:", tx.hash)
        await tx.wait()
        console.log("Cross-chain transfer completed")

        return true
      }
    } catch (error) {
      console.error("Cross-chain transfer error:", error)
      return false
    }
  }

  const checkBalance = async (amount: string) => {
    if (typeof window.ethereum !== "undefined") {
      try {
        const provider = new ethers.BrowserProvider(window.ethereum)
        const signer = await provider.getSigner()
        const address = await signer.getAddress()
        const balance = await provider.getBalance(address)
        const requiredAmount = ethers.parseEther(amount)

        if (balance < requiredAmount) {
          alert(
            `Insufficient balance. Required: ${amount} ${network === "Sepolia" ? "ETH" : "TCRO"}, Available: ${ethers.formatEther(balance)}`,
          )
          return false
        }
        return true
      } catch (error) {
        console.error("Balance check error:", error)
        return false
      }
    }
    return false
  }

  const startSwap = async () => {
    if (!secret.trim()) {
      alert("Please enter a secret")
      return
    }

    const isEthToTcro = fromToken === "ETH" && toToken === "TCRO"
    const amount = isEthToTcro ? "0.001" : "42.02"

    // Check balance before starting
    const hasBalance = await checkBalance(amount)
    if (!hasBalance) {
      return
    }

    setSwapStarted(true)
    const currentSteps = isEthToTcro ? ethToTcroSteps : tcroToEthSteps
    setSteps([...currentSteps])

    // Check and switch network if needed
    const requiredNetwork = isEthToTcro ? "sepolia" : "cronos"
    const currentNetworkName = isEthToTcro ? "Sepolia" : "Cronos"

    if (network !== currentNetworkName) {
      await switchNetwork(requiredNetwork)
      // Continue with swap after network switch
      await new Promise((resolve) => setTimeout(resolve, 2000)) // Wait for network switch
    }

    // Execute first 4 steps over 25 seconds (6.25 seconds each)
    for (let i = 0; i < 4; i++) {
      await new Promise((resolve) => setTimeout(resolve, 6250))
      setSteps((prev) =>
        prev.map((step, index) =>
          index === i
            ? { ...step, completed: true, current: false }
            : index === i + 1
              ? { ...step, current: true }
              : step,
        ),
      )
      setCurrentStep(i + 1)
    }

    setShowRevealButton(true)
  }

  const revealSecret = async () => {
    setShowRevealButton(false)
    setIsExecutingContract(true)

    try {
      // Step 5: Cross-chain transfer initiated (2 seconds)
      await new Promise((resolve) => setTimeout(resolve, 2000))
      setSteps((prev) => prev.map((step, index) => (index === 4 ? { ...step, completed: true, current: false } : step)))

      // Show claim funds button instead of executing immediately
      setShowClaimButton(true)
      setIsExecutingContract(false)
    } catch (error) {
      console.error("Reveal secret error:", error)
      setShowRevealButton(true)
      setIsExecutingContract(false)
    }
  }

  const claimFunds = async () => {
    setShowClaimButton(false)
    setIsExecutingContract(true)

    const isEthToTcro = fromToken === "ETH" && toToken === "TCRO"

    try {
      // Execute the cross-chain transfer (this will trigger MetaMask popup)
      const success = await executeCrossChainTransfer(isEthToTcro)

      if (success) {
        // Complete final steps over 8 seconds (4 seconds each)
        for (let i = 5; i < 7; i++) {
          await new Promise((resolve) => setTimeout(resolve, 4000))
          setSteps((prev) =>
            prev.map((step, index) => (index === i ? { ...step, completed: true, current: false } : step)),
          )
        }
        setSwapCompleted(true)
      } else {
        // Reset if transaction failed
        setShowClaimButton(true)
      }
    } catch (error) {
      console.error("Claim funds error:", error)
      setShowClaimButton(true)
    }

    setIsExecutingContract(false)
  }

  const handleSwap = () => {
    if (!isConnected) {
      alert("Please connect your wallet first")
      return
    }

    if (fromAmount === "0.0" || !fromAmount) {
      alert("Please enter an amount to swap")
      return
    }

    setShowSwapModal(true)
  }

  const resetSwap = () => {
    setShowSwapModal(false)
    setSecret("")
    setSwapStarted(false)
    setShowRevealButton(false)
    setShowClaimButton(false)
    setSwapCompleted(false)
    setCurrentStep(0)
    setSteps(fromToken === "ETH" ? ethToTcroSteps : tcroToEthSteps)
  }

  useEffect(() => {
    if (fromToken === "ETH" && toToken === "TCRO") {
      setToAmount((Number.parseFloat(fromAmount) * 26430.27).toFixed(2))
    } else if (fromToken === "TCRO" && toToken === "ETH") {
      setToAmount((Number.parseFloat(fromAmount) / 26430.27).toFixed(6))
    }
  }, [fromAmount, fromToken, toToken])

  const swapTokens = () => {
    const tempToken = fromToken
    const tempAmount = fromAmount
    setFromToken(toToken)
    setToToken(tempToken)
    setFromAmount(toAmount)
    setToAmount(tempAmount)
  }

  // Check for existing connection on mount
  useEffect(() => {
    const checkConnection = async () => {
      if (typeof window.ethereum !== "undefined") {
        try {
          const accounts = await window.ethereum.request({ method: "eth_accounts" })
          if (accounts.length > 0) {
            setAccount(accounts[0])
            setIsConnected(true)
            const chainId = await window.ethereum.request({ method: "eth_chainId" })
            updateNetworkStatus(chainId)

            // Set up event listeners for existing connection
            window.ethereum.on("accountsChanged", handleAccountsChanged)
            window.ethereum.on("chainChanged", handleChainChanged)
            window.ethereum.on("disconnect", handleDisconnect)
          }
        } catch (error) {
          console.error("Error checking existing connection:", error)
        }
      }
    }

    checkConnection()
  }, [])

  // Cleanup event listeners on unmount
  useEffect(() => {
    return () => {
      if (typeof window.ethereum !== "undefined") {
        try {
          window.ethereum.removeAllListeners("accountsChanged")
          window.ethereum.removeAllListeners("chainChanged")
          window.ethereum.removeAllListeners("connect")
          window.ethereum.removeAllListeners("disconnect")
        } catch (error) {
          console.log("Cleanup error:", error)
        }
      }
    }
  }, [])

  return (
    <div className="min-h-screen bg-black relative overflow-hidden">
      {/* Enhanced Starfield Background with Twinkling */}
      <div className="absolute inset-0">
        {Array.from({ length: 150 }).map((_, i) => (
          <div
            key={i}
            className="absolute bg-white rounded-full opacity-60 animate-pulse"
            style={{
              left: `${Math.random() * 100}%`,
              top: `${Math.random() * 100}%`,
              width: `${Math.random() * 3 + 1}px`,
              height: `${Math.random() * 3 + 1}px`,
              animationDelay: `${Math.random() * 3}s`,
              animationDuration: `${Math.random() * 2 + 1}s`,
            }}
          />
        ))}
        {/* Larger twinkling stars */}
        {Array.from({ length: 50 }).map((_, i) => (
          <div
            key={`large-${i}`}
            className="absolute bg-white rounded-full opacity-80 animate-ping"
            style={{
              left: `${Math.random() * 100}%`,
              top: `${Math.random() * 100}%`,
              width: `${Math.random() * 2 + 2}px`,
              height: `${Math.random() * 2 + 2}px`,
              animationDelay: `${Math.random() * 4}s`,
              animationDuration: `${Math.random() * 3 + 2}s`,
            }}
          />
        ))}
      </div>

      {/* Header */}
      <div className="relative z-10 flex justify-between items-center p-6">
        <div className="flex items-center gap-2">
          <span className="text-sm text-gray-400">Powered by</span>
          <div className="bg-gray-800 px-3 py-1 rounded-full shadow-lg">
            <span className="text-white text-sm font-medium">1inch</span>
          </div>
        </div>

        <div className="flex items-center gap-2">
          {isConnected && (
            <>
              <Button
                variant="outline"
                size="sm"
                className="bg-black border-gray-600 text-white hover:bg-gray-900 shadow-lg"
                onClick={fullRefresh}
                title="Full page refresh"
              >
                <RefreshCw className="w-4 h-4" />
              </Button>
              <Button
                variant="outline"
                size="sm"
                className="bg-black border-gray-600 text-white hover:bg-gray-900 shadow-lg"
                onClick={disconnectWallet}
                title="Disconnect wallet"
              >
                <LogOut className="w-4 h-4" />
              </Button>
            </>
          )}
          <Button
            variant="outline"
            className="bg-black border-gray-600 text-white hover:bg-gray-900 shadow-lg"
            onClick={connectWallet}
            disabled={isConnected || isConnecting}
          >
            <Wallet className="w-4 h-4 mr-2" />
            {isConnecting
              ? "Connecting..."
              : isConnected
                ? `${account.slice(0, 6)}...${account.slice(-4)}`
                : "Connect Wallet"}
          </Button>
        </div>
      </div>

      {/* Network Status */}
      {isConnected && (
        <div className="relative z-10 text-center">
          <span className="text-gray-400 text-sm">Connected to: </span>
          <span className="text-white text-sm font-medium">{network}</span>
          {network === "Unknown" && (
            <div className="mt-2 space-x-2">
              <Button
                size="sm"
                variant="outline"
                className="bg-black border-gray-600 text-white hover:bg-gray-900 shadow-lg"
                onClick={() => switchNetwork("sepolia")}
              >
                Switch to Sepolia
              </Button>
              <Button
                size="sm"
                variant="outline"
                className="bg-black border-gray-600 text-white hover:bg-gray-900 shadow-lg"
                onClick={() => switchNetwork("cronos")}
              >
                Switch to Cronos
              </Button>
            </div>
          )}
        </div>
      )}

      {/* Main Content */}
      <div className="relative z-10 flex flex-col items-center justify-center min-h-[calc(100vh-200px)] px-4">
        <div className="text-center mb-12">
          {/* Logo */}
          <div className="mb-4 flex justify-center">
            <Image
              src="/images/interchainx-logo.png"
              alt="INTERCHAINX"
              width={600}
              height={150}
              className="max-w-full h-auto drop-shadow-2xl"
              priority
            />
          </div>
          <p className="text-gray-400 text-lg italic">"Because your crypto shouldn't be stuck in one galaxy."</p>
        </div>

        {/* Floating Swap Card */}
        <Card className="w-full max-w-md bg-gray-900/90 border-gray-700 backdrop-blur-sm shadow-2xl hover:shadow-3xl transition-all duration-300 transform hover:-translate-y-1">
          <CardContent className="p-6">
            <div className="text-center mb-6">
              <h2 className="text-2xl font-bold text-white mb-2">Cross-Chain Swap</h2>
              <p className="text-gray-400 text-sm">Rate: 1 ETH = 26430.27 TCRO</p>
            </div>

            <div className="space-y-4">
              {/* From Section */}
              <div>
                <label className="text-gray-400 text-sm mb-2 block">From</label>
                <div className="flex gap-2">
                  <Select value={fromToken} onValueChange={setFromToken}>
                    <SelectTrigger className="w-24 bg-gray-800 border-gray-600 text-white shadow-lg">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent className="bg-gray-800 border-gray-600 shadow-2xl">
                      <SelectItem value="ETH" className="text-white">
                        ETH
                      </SelectItem>
                      <SelectItem value="TCRO" className="text-white">
                        TCRO
                      </SelectItem>
                    </SelectContent>
                  </Select>
                  <Input
                    value={fromAmount}
                    onChange={(e) => setFromAmount(e.target.value)}
                    className="flex-1 bg-gray-800 border-gray-600 text-white placeholder-gray-500 shadow-lg"
                    placeholder="0.0"
                  />
                </div>
              </div>

              {/* Swap Icon */}
              <div className="flex justify-center">
                <Button
                  variant="ghost"
                  size="icon"
                  className="rounded-full bg-gray-800 hover:bg-gray-700 text-white shadow-lg hover:shadow-xl transition-all duration-300"
                  onClick={swapTokens}
                >
                  <ArrowUpDown className="w-4 h-4" />
                </Button>
              </div>

              {/* To Section */}
              <div>
                <label className="text-gray-400 text-sm mb-2 block">To</label>
                <div className="flex gap-2">
                  <Select value={toToken} onValueChange={setToToken}>
                    <SelectTrigger className="w-24 bg-gray-800 border-gray-600 text-white shadow-lg">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent className="bg-gray-800 border-gray-600 shadow-2xl">
                      <SelectItem value="ETH" className="text-white">
                        ETH
                      </SelectItem>
                      <SelectItem value="TCRO" className="text-white">
                        TCRO
                      </SelectItem>
                    </SelectContent>
                  </Select>
                  <Input
                    value={toAmount}
                    readOnly
                    className="flex-1 bg-gray-800 border-gray-600 text-white placeholder-gray-500 shadow-lg"
                    placeholder="0.0"
                  />
                </div>
              </div>

              {/* Swap Button */}
              <Button
                className="w-full bg-black hover:bg-gray-900 text-white mt-6 shadow-xl hover:shadow-2xl transition-all duration-300 transform hover:-translate-y-0.5"
                onClick={handleSwap}
                disabled={!isConnected || fromAmount === "0.0" || !fromAmount}
              >
                {!isConnected ? "Connect Wallet First" : "Swap"}
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* Enhanced Swap Modal */}
      <Dialog open={showSwapModal} onOpenChange={setShowSwapModal}>
        <DialogContent className="bg-gray-900/95 border-gray-700 text-white max-w-md shadow-2xl backdrop-blur-sm">
          <DialogHeader>
            <DialogTitle className="text-center">
              Cross-Chain Swap: {fromToken} → {toToken}
            </DialogTitle>
          </DialogHeader>

          <div className="space-y-6">
            {!swapStarted && (
              <div className="space-y-4">
                <div>
                  <label className="text-gray-400 text-sm mb-2 block">Enter Secret</label>
                  <Input
                    value={secret}
                    onChange={(e) => setSecret(e.target.value)}
                    className="bg-gray-800 border-gray-600 text-white shadow-lg"
                    placeholder="Enter your secret..."
                    type="password"
                  />
                </div>
                <Button
                  onClick={startSwap}
                  className="w-full bg-black hover:bg-gray-900 text-white shadow-xl hover:shadow-2xl transition-all duration-300"
                  disabled={!secret.trim()}
                >
                  Start Swap
                </Button>
              </div>
            )}

            {swapStarted && (
              <div className="space-y-4">
                <div className="space-y-3">
                  {steps.map((step, index) => (
                    <div key={step.id} className="flex items-center gap-3">
                      <div
                        className={`w-6 h-6 rounded-full flex items-center justify-center shadow-lg ${
                          step.completed ? "bg-gray-700" : step.current ? "bg-gray-600" : "bg-gray-800"
                        }`}
                      >
                        {step.completed ? (
                          <Check className="w-4 h-4 text-white" />
                        ) : step.current ? (
                          <Clock className="w-4 h-4 animate-spin text-white" />
                        ) : (
                          <span className="text-xs text-white">{step.id}</span>
                        )}
                      </div>
                      <span
                        className={`text-sm ${
                          step.completed ? "text-white" : step.current ? "text-gray-300" : "text-gray-500"
                        }`}
                      >
                        {step.title}
                      </span>
                    </div>
                  ))}
                </div>

                {showRevealButton && (
                  <Button
                    onClick={revealSecret}
                    className="w-full bg-black hover:bg-gray-900 text-white shadow-xl hover:shadow-2xl transition-all duration-300"
                    disabled={isExecutingContract}
                  >
                    <Eye className="w-4 h-4 mr-2" />
                    {isExecutingContract ? "Executing Cross-Chain Transfer..." : "Reveal Secret"}
                  </Button>
                )}

                {showClaimButton && (
                  <Button
                    onClick={claimFunds}
                    className="w-full bg-black hover:bg-gray-900 text-white shadow-xl hover:shadow-2xl transition-all duration-300"
                    disabled={isExecutingContract}
                  >
                    <Wallet className="w-4 h-4 mr-2" />
                    {isExecutingContract ? "Processing Transaction..." : "Claim Funds"}
                  </Button>
                )}

                {swapCompleted && (
                  <div className="text-center space-y-4">
                    <div className="text-white font-medium">Cross-Chain Swap Completed Successfully!</div>
                    <div className="text-gray-400 text-sm">TCRO ↔ ETH transfers completed between networks</div>
                    <Button
                      onClick={resetSwap}
                      className="w-full bg-black hover:bg-gray-900 text-white shadow-xl hover:shadow-2xl transition-all duration-300"
                    >
                      Close
                    </Button>
                  </div>
                )}
              </div>
            )}
          </div>
        </DialogContent>
      </Dialog>

      {/* Footer */}
      <div className="relative z-10 text-center pb-6">
        <div className="text-gray-400 text-sm">
          <span>Built at </span>
          <span className="text-white">Unite Defi</span>
          <span> | </span>
          <span className="text-white">EthGlobal</span>
        </div>
        <div className="text-gray-500 text-xs mt-1">
          <span>Sepolia Testnet</span>
          <span className="mx-2">•</span>
          <span>Cronos Testnet</span>
        </div>
      </div>

      <style jsx>{`
        @keyframes twinkle {
          0%, 100% { opacity: 0.3; transform: scale(1); }
          50% { opacity: 1; transform: scale(1.2); }
        }
        
        .animate-twinkle {
          animation: twinkle 2s ease-in-out infinite;
        }
      `}</style>
    </div>
  )
}
