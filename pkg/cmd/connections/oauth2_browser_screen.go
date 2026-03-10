/*
 * Copyright (c) 2026 Versori Group Inc
 *
 * Use of this software is governed by the Business Source License 1.1
 * included in the LICENSE file at the root of this repository.
 *
 * Change Date: 2030-03-01
 * Change License: Apache License, Version 2.0
 *
 * As of the Change Date, in accordance with the Business Source License,
 * use of this software will be governed by the Apache License, Version 2.0.
 */

package connections

const oauth2Screen = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>OAuth Authentication Successful</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: linear-gradient(135deg, #0C1B24 0%, #052A3D 100%);
            display: flex;
            justify-content: center;
            align-items: center;
            min-height: 100vh;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 16px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
            padding: 48px;
            text-align: center;
            max-width: 480px;
            width: 100%;
            animation: slideIn 0.4s ease-out;
        }
        @keyframes slideIn {
            from {
                opacity: 0;
                transform: translateY(-20px);
            }
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }
        .checkmark {
            width: 80px;
            height: 80px;
            border-radius: 50%;
            display: inline-block;
            background: #10b981;
            position: relative;
            margin-bottom: 24px;
            animation: pulse 0.6s ease-out;
        }
        @keyframes pulse {
            0% { transform: scale(0); opacity: 0; }
            50% { transform: scale(1.1); }
            100% { transform: scale(1); opacity: 1; }
        }
        .checkmark::after {
            content: '';
            position: absolute;
            width: 24px;
            height: 40px;
            border: solid white;
            border-width: 0 4px 4px 0;
            top: 14px;
            left: 28px;
            transform: rotate(45deg);
        }
        h1 {
            color: #1f2937;
            font-size: 28px;
            font-weight: 700;
            margin-bottom: 16px;
        }
        p {
            color: #6b7280;
            font-size: 16px;
            line-height: 1.6;
            margin-bottom: 12px;
        }
        .emphasis {
            color: #4b5563;
            font-weight: 600;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="checkmark"></div>
        <h1>Authentication Successful!</h1>
        <p>Your OAuth connection has been authorized successfully.</p>
        <p class="emphasis">You can now close this window and return to your terminal.</p>
    </div>
</body>
</html>`
