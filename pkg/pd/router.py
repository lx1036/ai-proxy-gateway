

import argparse
import json
import time
from dataclasses import dataclass
from typing import Optional

from fastapi import FastAPI, Request
from contextlib import asynccontextmanager
import httpx
from fastapi.responses import StreamingResponse,JSONResponse


@dataclass
class ClientInfo:
    client: httpx.AsyncClient
    host: Optional[str] = None
    init_port: Optional[int] = None
    alloc_port: Optional[int] = None

@asynccontextmanager
async def lifespan(app: FastAPI):
    prefill_hosts = global_args.prefiller_host
    prefill_ports = global_args.prefiller_port

    prefiller_base_url = f"http://{prefill_hosts[0]}:{int(prefill_ports[0])}"
    prefill_client = httpx.AsyncClient(timeout=None, base_url=prefiller_base_url)
    app.state.prefill_clients.append(
        ClientInfo(client=prefill_client)
    )

    decode_hosts = global_args.decoder_host
    decode_ports = global_args.decoder_port
    decoder_base_url = f"http://{decode_hosts[0]}:{int(decode_ports[0])}"
    decode_client = httpx.AsyncClient(timeout=None, base_url=decoder_base_url)
    app.state.decode_clients.append(
        ClientInfo(
            decode_client,
            decode_hosts[0],
            global_args.decoder_init_port,
            global_args.decoder_alloc_port,
        )
    )

    app.state.total_clients = app.state.prefill_clients + app.state.decode_clients

    yield


# Update FastAPI app initialization to use lifespan
app = FastAPI(lifespan=lifespan)
# Initialize variables to hold the persistent clients
app.state.prefill_clients = []
app.state.decode_clients = []
app.state.total_clients = []

async def send_request_to_service(
        client: httpx.AsyncClient, endpoint: str, req_data: dict
):
    """
    Send a request to a service using a persistent client.
    """

    headers = {"Authorization": f"Bearer abc"}
    response = await client.post(endpoint, json=req_data, headers=headers)
    response.raise_for_status()
    return response


async def stream_service_response(
        client: httpx.AsyncClient, endpoint: str, req_data: dict
):
    """
    Asynchronously stream the response from a service using a persistent client.
    """
    headers = {"Authorization": f"Bearer abc"}
    async with client.stream(
            "POST", endpoint, json=req_data, headers=headers
    ) as response:
        response.raise_for_status()
        async for chunk in response.aiter_bytes():
            yield chunk

# for Nixl
@app.post("/v1/chat/completions")
async def handle_chat_completions(request: Request):
    try:
        req_data = await request.json()

        tokenization_client = app.state.prefill_clients[0]
        tokenize_output = await send_request_to_service(
            tokenization_client.client, "/tokenize", {"messages": req_data["messages"]}
        )
        tokenize_output = tokenize_output.json()

# Tokenize output: {'count': 20, 'max_model_len': 32768, 'tokens': [151644, 8948, 198, 2610, 525, 264, 10950, 17847, 13, 151645, 198, 151644, 872, 198, 108386, 151645, 198, 151644, 77091, 198]}
# Tokenize output: {'count': 20, 'max_model_len': 32768, 'tokens': [151644, 8948, 198, 2610, 525, 264, 10950, 17847, 13, 151645, 198, 151644, 872, 198, 108386, 151645, 198, 151644, 77091, 198], 'token_strs': None}
        print("Tokenize output:", tokenize_output)

        org_max_tokens = req_data["max_tokens"]
        req_data["prompt"] = tokenize_output["tokens"]
        req_data["max_tokens"] = 1

        org_max_completion_tokens = None
        if "max_completion_tokens" in req_data:
            org_max_completion_tokens = req_data["max_completion_tokens"]
            req_data["max_completion_tokens"] = 1

        # Pick decode client
        decode_client = app.state.decode_clients[0]
        disagg_spec = {
            "req_id": "123",
            "receiver_host": decode_client.host,
            "receiver_init_port": decode_client.init_port,
            "receiver_alloc_port": decode_client.alloc_port,
        }
        req_data["kv_transfer_params"] = {
            "ret_first_tok": True,
            "disagg_spec": disagg_spec,
            "do_remote_decode": True,
            "do_remote_prefill": False,
        }
        req_data["stream"] = False
        stream_options = req_data.pop("stream_options", None)


        # send request to prefill pod
        prefill_client = app.state.prefill_clients[0]

        prefill_output = await send_request_to_service(
            prefill_client.client, "/v1/completions", req_data
        )
        prefill_output = prefill_output.json()
# Prefill output: {'id': 'cmpl-9698e1bbe64e4d0aa3838e1fefc4eeb9', 'object': 'text_completion', 'created': 1776157718,
# 'model': 'qwen', 'choices': [{'index': 0, 'text': '你好', 'logprobs': None, 'finish_reason': 'length', 'stop_reason': None,
# 'prompt_logprobs': None}], 'usage': {'prompt_tokens': 20, 'total_tokens': 21, 'completion_tokens': 1, 'prompt_tokens_details': None}}

# 使用 NixlConnectorV1 不可以
# Prefill output: {'id': 'cmpl-a6e9ec5d2fdd43a2b8608c8426490d78', 'object': 'text_completion', 'created': 1776245762, 'model': 'qwen',
        # 'choices': [{'index': 0, 'text': '你好', 'logprobs': None, 'finish_reason': 'length', 'stop_reason': None, 'token_ids': None,
        # 'prompt_logprobs': None, 'prompt_token_ids': None}], 'service_tier': None, 'system_fingerprint': None, 'usage':
        # {'prompt_tokens': 20, 'total_tokens': 21, 'completion_tokens': 1, 'prompt_tokens_details': None}, 'kv_transfer_params': None}

# 使用 LMCacheConnectorV1 是可以的
# Prefill output: {'id': 'cmpl-8565d4ffaf5e49d8887dfb29eba53d0b', 'object': 'text_completion', 'created': 1776246089, 'model': 'qwen', 'choices':
        # [{'index': 0, 'text': '你好', 'logprobs': None, 'finish_reason': 'length', 'stop_reason': None, 'token_ids': None, 'prompt_logprobs': None,
        # 'prompt_token_ids': None}], 'service_tier': None, 'system_fingerprint': None, 'usage': {'prompt_tokens': 20, 'total_tokens': 21,
        # 'completion_tokens': 1, 'prompt_tokens_details': None}, 'kv_transfer_params': {'first_tok': 108386}}

# 使用 NixlConnector 可以。这里配置错了，导致 'remote_host': '0.0.0.0'
# Prefill output: {'id': 'cmpl-3d141591ce944492be9faa0bab71130e', 'object': 'text_completion', 'created': 1776259596, 'model': 'qwen',
        # 'choices': [{'index': 0, 'text': '我是', 'logprobs': None, 'finish_reason': 'length', 'stop_reason': None, 'token_ids': None,
        # 'prompt_logprobs': None, 'prompt_token_ids': None}], 'service_tier': None, 'system_fingerprint': None, 'usage':
        # {'prompt_tokens': 25, 'total_tokens': 26, 'completion_tokens': 1, 'prompt_tokens_details': None},
        # 'kv_transfer_params': {'do_remote_prefill': True, 'do_remote_decode': False, 'remote_block_ids': [1, 2], 'remote_engine_id': '286f9ee1-62bb-4038-8cec-91ed1eb30d29', 'remote_host': '0.0.0.0', 'remote_port': 5557, 'tp_size': 1}}


# Prefill output: {'id': 'cmpl-1bd90d0edaf9465ab132bf1d45e70218', 'object': 'text_completion', 'created': 1776259942, 'model': 'qwen', 'choices': [{'index': 0, 'text': '我是', 'logprobs': None, 'finish_reason': 'length', 'stop_reason': None, 'token_ids': None, 'prompt_logprobs': None, 'prompt_token_ids': None}], 'service_tier': None, 'system_fingerprint': None, 'usage': {'prompt_tokens': 25, 'total_tokens': 26, 'completion_tokens': 1, 'prompt_tokens_details': None},
        # 'kv_transfer_params': {'do_remote_prefill': True, 'do_remote_decode': False, 'remote_block_ids': [3, 4], 'remote_engine_id': '286f9ee1-62bb-4038-8cec-91ed1eb30d29', 'remote_host': '0.0.0.0', 'remote_port': 5557, 'tp_size': 1}}


# 使用 NixlConnector，但是没有 kv_transfer_params.first_tok 字段
# Prefill output:
#         {'id': 'cmpl-03b3d7e20504401b80fd9dc9f5730b9b', 'object': 'text_completion', 'created': 1776334003, 'model': 'qwen',
#         'choices': [{'index': 0, 'text': '我是', 'logprobs': None, 'finish_reason': 'length', 'stop_reason': None, 'token_ids': None, 'prompt_logprobs': None, 'prompt_token_ids': None}],
#         'service_tier': None, 'system_fingerprint': None, 'usage': {'prompt_tokens': 25, 'total_tokens': 26, 'completion_tokens': 1, 'prompt_tokens_details': None},
#         'kv_transfer_params': {'do_remote_prefill': True, 'do_remote_decode': False, 'remote_block_ids': [1, 2], 'remote_engine_id': 'f2e8e2e0-e662-4568-a07b-c05a1e5e4ee5',
#         'remote_host': '10.243.20.29', 'remote_port': 5557, 'tp_size': 1}}
        print("Prefill output:", prefill_output)

        # wait_decode_kv_ready()
        time.sleep(1)

        req_data["max_tokens"] = org_max_tokens - 1
        if org_max_completion_tokens is not None:
            req_data["max_completion_tokens"] = org_max_completion_tokens - 1
        # Add the first token from prefill to the tokenized messages for decode
        # req_data["prompt"].append(prefill_output["kv_transfer_params"]["first_tok"])
        req_data["include_usage"] = True
        req_data.pop("kv_transfer_params")
        req_data["kv_transfer_params"] = prefill_output["kv_transfer_params"]
        req_data["stream"] = True
        if stream_options is not None:
            req_data["stream_options"] = stream_options

        # Stream response from decode service
        async def generate_stream():
            head_chunk = {
                "id": prefill_output["id"],
                "object": "text_completion",
                "created": prefill_output["created"],
                "model": prefill_output["model"],
                "choices": [
                    {
                        "index": 0,
                        "text": prefill_output["choices"][0]["text"],
                        "logprobs": None,
                        "finish_reason": None,
                        "stop_reason": None,
                    }
                ],
                "usage": None,
            }
            yield ("data: " + json.dumps(head_chunk, separators=(",", ":")) + "\n\n").encode()

            # Wait until decode node signals that kv is ready
            # await wait_decode_kv_ready(req_id, num_tp_rank)

            async for chunk in stream_service_response(decode_client.client, "/v1/completions", req_data):
                yield chunk

        # decode_response = await send_request_to_service(
        #     decode_client.client, "/v1/completions", req_data
        # )
        # decode_response = decode_response.json()
        # print("Decode response:", decode_response)
        # decode_response = decode_response.json()
        # return  JSONResponse(decode_response.json(), media_type="application/json")
        return StreamingResponse(generate_stream(), media_type="application/json")



    except Exception as e:
        # Standard
        import sys
        import traceback
        exc_info = sys.exc_info()
        print(
            "Error occurred in disagg prefill proxy server  - chat completions endpoint"
        )
        print(e)
        print("".join(traceback.format_exception(*exc_info)))
        raise





def csv_ints(s):
    return [int(x) for x in s.split(",")]


def csv_strs(s):
    return [x.strip() for x in s.split(",")]

def parse_args():
    parser = argparse.ArgumentParser()

    parser.add_argument("--port", type=int, default=8000)
    parser.add_argument("--host", type=str, default="localhost")
    parser.add_argument("--prefiller-host", type=csv_strs, default=["localhost"])
    parser.add_argument("--prefiller-port", type=csv_ints, default=[7100])
    parser.add_argument("--num-prefillers", type=int, default=1)
    parser.add_argument("--decoder-host", type=csv_strs, default=["localhost"])
    parser.add_argument("--decoder-port", type=csv_ints, default=[7200])
    parser.add_argument("--decoder-init-port", type=csv_ints, default=[8300])
    parser.add_argument("--decoder-alloc-port", type=csv_ints, default=[8400])

    parser.add_argument("--num-decoders", type=int, default=1)
    parser.add_argument("--proxy-host", type=str, default="localhost")
    parser.add_argument("--proxy-port", type=int, default=8500)

    args = parser.parse_args()
    return args

if __name__ == "__main__":
    global global_args
    global_args = parse_args()

    # Third Party
    import uvicorn

    uvicorn.run(app, host=global_args.host, port=global_args.port)



"""
### LMCache:
k port-forward -n envoy-gateway-system pod/vllm-1p1d-roleset-vhzgl-prefill-cbfc9f574-0 7100:8000

k port-forward -n envoy-gateway-system pod/vllm-1p1d-roleset-vhzgl-decode-7bcfccd697-0 7200:8000


python3 ./router.py \
        --host localhost \
        --port 8000 \
        --prefiller-host localhost \
        --prefiller-port 7100 \
        --num-prefillers 1 \
        --decoder-host localhost \
        --decoder-port 7200  \
        --decoder-init-port 7300 \
        --decoder-alloc-port 7400 \
        --proxy-host localhost \
        --proxy-port 7500 \
        --num-decoders 1


curl -H "Accept: application/json" \
     -H "Content-type: application/json" \
     -X POST \
     -d '{
         "model": "qwen",
         "stream": false,
         "messages": [
              {"role": "user", "content": "你好"}
          ],
          "max_tokens": 50
     }' \
     http://127.0.0.1:8000/v1/chat/completions


### Nixl:


"""
