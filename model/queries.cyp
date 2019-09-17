// Useful queries

// Find all consents given by an Identity
MATCH (i:Identity {sub:$sub})
MATCH (i)<-[:GrantedBy]-(r:Rule)-[:Grant]->(p:Policy)-[:Grant]->(a:Permission)
MATCH (n:Identity)-[:IsGranted]->(r)
return i, r, n, p, a

// Find all scopes exposed by an identity
MATCH (i:Identity {sub:$sub})-[:Exposes]->(p:Policy)-[:Grant]->(a:Permission)
return i, p, a


// Create invite
MATCH (i:Identity {email:"mnk@fullrate.dk"})
MERGE (i)-[:INVITES]-(inv:Identity:Invite {id:randomUUID()})-[:SENT_TO]-(:Email {email:"snk@cybertron.dk"})

WITH i, inv

// TODO: Add EXPOSE PART OF SEARCH OR SUFFER!
MATCH (p:Permission {name:"update:identity"})
MERGE (inv)-[:IS_GRANTED]->(gr:GrantRule)-[:GRANT]->(p)

WITH i, inv, gr, p

MERGE (inv)-[:FOLLOWS]->(i)

return i, inv, gr, p


// Accept invite
MATCH (inv:Identity:Invite {id:"02660407-ebdd-469b-ab15-56e052f0cb91"})
// Authenticated Identity
MATCH (i:Identity {id:"c704b592-9b84-48ec-a8af-f26a8ae1b2bc"})

WITH inv, i

// Attach relation ship to authenticated node
MATCH (inv)-[r:IS_GRANTED]->(gr:GrantRule)
MERGE (i)-[:IS_GRANTED]->(gr)
DELETE r

WITH inv, i

MATCH (inv)-[r:FOLLOWS]-(n:Identity)
MERGE (i)-[:FOLLOWS]->(n)
DELETE r

WITH inv, i

MATCH (inv)-[r:SENT_TO]-(n:Email)
MERGE (i)-[:SENT_TO]->(n)
DELETE r

WITH inv, i

MATCH (inv)<-[r:INVITES]-(n:Identity)
MERGE (i)<-[:INVITES]->(n)
DELETE r

WITH inv, i

DELETE (inv)

return inv, i