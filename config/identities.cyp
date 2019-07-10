CREATE CONSTRAINT ON (i:Identity) ASSERT i.sub IS UNIQUE;
CREATE CONSTRAINT ON (i:Identity) ASSERT i.email IS UNIQUE;
CREATE (i:Identity {sub: "user-1", password: "$2y$12$W8wsynt8t9RBlyHa4Z0R../zgJxlGH6rQhjYHbUCFtFEVdO87YgzK", name:"User 1", email:"user-1@domain.com"});
CREATE (i:Identity {sub: "wraix", password: "$2y$12$Y8xtFTYTgJIgycdSAwhXsePvdUsndX6bOR7zWidsE92YFEG1c/TuC", name:"Wraix", email:"wraix@domain.com"});
