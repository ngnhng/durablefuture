export function meta() {
  return [
    { title: "Environment - DurableFuture" },
    { name: "description", content: "Environment configuration" },
  ];
}

export default function Environment() {
  return (
    <div className="p-8">
      <div className="max-w-4xl mx-auto">
        <h1 className="text-2xl font-semibold text-white mb-8">Environment</h1>
        <div className="bg-gray-900 rounded-lg p-8">
          <p className="text-gray-400">Environment configuration and settings will be displayed here.</p>
        </div>
      </div>
    </div>
  );
}