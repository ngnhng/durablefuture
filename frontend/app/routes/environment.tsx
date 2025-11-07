/**
 * Copyright 2025 Nguyen Nhat Nguyen
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

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