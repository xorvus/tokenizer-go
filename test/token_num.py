import os
import tiktoken as tk

def read_data_from_file(filename):
    """
    Read data from test file
    :param filename: file name
    :return: text_list, model_list, encoding_list
    """
    if not os.path.exists(filename):
        filename = "test.txt"
    with open(filename, 'r', encoding='utf-8') as f:
        lines = f.read().splitlines()
        text_list = lines[0].split(',')
        model_list = lines[1].split(',')
        encoding_list = lines[2].split(',')
    
    return text_list, model_list, encoding_list

def get_token_by_model(text, model):
    """
    Get token count by model
    """
    try:
        encoding = tk.encoding_for_model(model)
        return len(encoding.encode(text))
    except Exception as e:
        print(f"Error model {model}: {e}")
        return 0

def get_token_by_encoding(text, encoding_name):
    """
    Get token count by encoding name
    """
    try:
        encoding = tk.get_encoding(encoding_name)
        return len(encoding.encode(text))
    except Exception as e:
        print(f"Error encoding {encoding_name}: {e}")
        return 0

def test_token_by_model(text_list, model_list):
    for text in text_list:
        for model in model_list:
            num_tokens = get_token_by_model(text, model)
            print(f"text: {text}, model: {model}, token: {num_tokens}")

def test_token_by_encoding(text_list, encoding_list):
    for text in text_list:
        for encoding in encoding_list:
            num_tokens = get_token_by_encoding(text, encoding)
            print(f"text: {text}, encoding: {encoding}, token: {num_tokens}")

if __name__ == '__main__':
    text_list, model_list, encoding_list = read_data_from_file('test/test.txt')
    test_token_by_model(text_list, model_list)
    print("=========================================")
    test_token_by_encoding(text_list, encoding_list)
