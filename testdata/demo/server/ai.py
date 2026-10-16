from openai import OpenAI

client = OpenAI()  # calls https://api.openai.com/v1


def summarize(text: str) -> str:
    r = client.chat.completions.create(model="gpt-4o-mini", messages=[{"role": "user", "content": text}])
    return r.choices[0].message.content
