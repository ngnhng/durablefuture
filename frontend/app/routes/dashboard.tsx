export function meta() {
  return [
    { title: "Dashboard - DurableFuture" },
    { name: "description", content: "Main dashboard" },
  ];
}

export default function Dashboard() {
  return (
    <div className="p-8">
      <div className="max-w-4xl mx-auto">
        {/* Header */}
        <div className="flex items-center justify-between mb-8">
          <h1 className="text-2xl font-semibold text-white">Tasks</h1>
          <div className="flex items-center gap-2 text-sm text-gray-400">
            <div className="w-4 h-4 bg-blue-500 rounded-full flex items-center justify-center">
              <span className="text-xs text-white">📖</span>
            </div>
            <span>Task docs</span>
          </div>
        </div>

        {/* Main Content */}
        <div className="bg-gray-900 rounded-lg p-8">
          <div className="text-center">
            <div className="flex items-center justify-center gap-2 mb-6">
              <h2 className="text-xl font-medium text-white">Get setup in 3 minutes</h2>
              <div className="flex items-center gap-1 text-sm text-gray-400">
                <span className="w-4 h-4 bg-blue-500 rounded-full flex items-center justify-center">
                  <span className="text-xs text-white">🧠</span>
                </span>
                <span>I'm stuck!</span>
              </div>
            </div>

            <div className="space-y-6 text-left max-w-2xl mx-auto">
              {/* Step 1 */}
              <div className="flex gap-4">
                <div className="flex-shrink-0 w-6 h-6 bg-blue-600 text-white rounded-full flex items-center justify-center text-sm font-medium">
                  1
                </div>
                <div className="flex-1">
                  <h3 className="text-white font-medium mb-2">
                    Run the CLI 'init' command in an existing project
                  </h3>
                  <div className="flex gap-2 text-sm mb-2">
                    <span className="px-2 py-1 bg-gray-800 text-blue-400 rounded">npm</span>
                    <span className="px-2 py-1 bg-gray-800 text-blue-400 rounded">pnpm</span>
                    <span className="px-2 py-1 bg-gray-800 text-blue-400 rounded">yarn</span>
                  </div>
                  <div className="bg-gray-800 rounded p-3 font-mono text-sm text-gray-300 flex items-center justify-between">
                    <span>npx trigger.dev@latest init -p proj_vileunzhdycdyalkgouc</span>
                    <button className="text-gray-400 hover:text-white">
                      📋
                    </button>
                  </div>
                  <p className="text-gray-400 text-sm mt-2">
                    You'll notice a new folder in your project called{" "}
                    <code className="text-blue-400">trigger</code>. We've added a few simple
                    example tasks in there to help you get started.
                  </p>
                </div>
              </div>

              {/* Step 2 */}
              <div className="flex gap-4">
                <div className="flex-shrink-0 w-6 h-6 bg-blue-600 text-white rounded-full flex items-center justify-center text-sm font-medium">
                  2
                </div>
                <div className="flex-1">
                  <h3 className="text-white font-medium mb-2">
                    Run the CLI 'dev' command
                  </h3>
                  <div className="flex gap-2 text-sm mb-2">
                    <span className="px-2 py-1 bg-gray-800 text-blue-400 rounded">npm</span>
                    <span className="px-2 py-1 bg-gray-800 text-blue-400 rounded">pnpm</span>
                    <span className="px-2 py-1 bg-gray-800 text-blue-400 rounded">yarn</span>
                  </div>
                  <div className="bg-gray-800 rounded p-3 font-mono text-sm text-gray-300 flex items-center justify-between">
                    <span>npx trigger.dev@latest dev</span>
                    <button className="text-gray-400 hover:text-white">
                      📋
                    </button>
                  </div>
                </div>
              </div>

              {/* Step 3 */}
              <div className="flex gap-4">
                <div className="flex-shrink-0 w-6 h-6 bg-gray-600 text-white rounded-full flex items-center justify-center text-sm font-medium">
                  3
                </div>
                <div className="flex-1">
                  <h3 className="text-white font-medium mb-2">
                    Waiting for tasks{" "}
                    <span className="inline-block w-4 h-4 border-2 border-blue-500 border-t-transparent rounded-full animate-spin ml-2"></span>
                  </h3>
                  <p className="text-gray-400 text-sm">
                    This page will automatically refresh.
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}