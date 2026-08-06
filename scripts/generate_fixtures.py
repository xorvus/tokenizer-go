#!/usr/bin/env python3
import json
import os
import tiktoken

CORPUS = [
    "",
    "hello",
    "hello world",
    "Hello, world!",
    "你好世界",
    "こんにちは",
    "😀",
    "hello\nworld",
    "Bahasa Indonesia adalah bahasa resmi Republik Indonesia.",
    "const x = 42;\nconsole.log(x);",
]

def generate_fixture(encoding_name, output_path):
    enc = tiktoken.get_encoding(encoding_name)
    cases = []
    for text in CORPUS:
        tokens = enc.encode_ordinary(text)
        cases.append({"text": text, "tokens": tokens})
    
    data = {
        "encoding": encoding_name,
        "reference_version": tiktoken.__version__,
        "cases": cases
    }
    
    os.makedirs(os.path.dirname(output_path), exist_ok=True)
    with open(output_path, "w", encoding="utf-8") as f:
        json.dump(data, f, ensure_ascii=False, indent=2)

if __name__ == "__main__":
    generate_fixture("cl100k_base", "test/cl100k_base.json")
    generate_fixture("o200k_base", "test/o200k_base.json")
