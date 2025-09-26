export function meta() {
  return [
    { title: "Deployments - DurableFuture" },
  ];
}

export default function Deployments() {
  return (
    <div className="p-8">
      <div className="max-w-4xl mx-auto">
        <h1 className="text-2xl font-semibold text-white mb-8">Deployments</h1>
        <div className="bg-gray-900 rounded-lg p-8">
          <p className="text-gray-400">Deployment management interface.</p>
        </div>
      </div>
    </div>
  );
}