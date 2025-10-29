from locust import FastHttpUser, task, between
import random

# Only include searchable terms: Name and categories
NAME_TERMS = ["product"]  # generic name keywords
CATEGORY_TERMS = ["electronics", "books", "home", "outdoors", "clothes"]

class ProductSearchUser(FastHttpUser):
    wait_time = lambda self: 0

    @task
    def search_products(self):
        if random.random() < 0.5:
            term = random.choice(NAME_TERMS)
            search_type = "Name"
        else:
            term = random.choice(CATEGORY_TERMS)
            search_type = "Category"

        response = self.client.get(f"/products/search?q={term}&debug=1")
        if response.status_code == 200:
            data = response.json()
            print(f"{search_type} search '{term}' → found {data.get('total_found')} products, "
                  f"checked {data.get('checked_request')} items, "
                  f"total checked {data.get('total_checked')}")
        else:
            print(f"Error {response.status_code} for search '{term}'")